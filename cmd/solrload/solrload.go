package main

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sync"
	"time"

	"github.com/akamensky/argparse"
	"github.com/frizner/glsolr"
)

const (
	// the program name and version
	name    = "solrload"
	version = "0.1.0"

	// RE to check web link to a Solr collection
	reLink = `^((https|http):\/\/((([A-Za-z0-9-.]+:\d+)|([a-z0-9-.]+))|([a-z0-9-]+))\/solr\/([\\._A-Za-z0-9\\-]+|([\\._A-Za-z0-9\\-]+)\/))$`

	// the names of environment variables to get the username and password if they aren't set in the command line
	userEnv  = "SOLRUSER"
	passwEnv = "SOLRPASSW"

	// default settings

	// http timeout in seconds by default
	dftHTTPTimeout = 180

	// Number of updating queries in parallel
	dftNQueries = 8
)

type params struct {
	cLink, srcDir, user, passw string
	nQueries, httpTimeout      int
	commit                     bool
}

type infoMsg struct {
	fileName string
	err      error
}

// Parse arguments
func parceArgs(name, lMask string, args []string) (p *params, err error) {
	parsHelp := fmt.Sprintf("%s uploads documents from JSON files in a Solr collection (index) using the update queries in parallel", name)
	parser := argparse.NewParser(name, fmt.Sprintf("%s ", parsHelp))
	cLink := parser.String("c", "collink", &argparse.Options{Required: true,
		Help: "http link to a Solr collection like http[s]://address[:port]/solr/collection"})

	nQueries := parser.Int("n", "nqueries", &argparse.Options{Required: false, Default: dftNQueries,
		Help: "Number of updating queries in parallel"})

	srcDir := parser.String("s", "src", &argparse.Options{Required: false, Default: ".",
		Help: "Path to the dump directory with JSON files to upload"})

	nocommit := parser.NewCommand("--nocommit", "Won't do the commit after the each update query")

	user := parser.String("u", "user", &argparse.Options{Required: false, Default: "",
		Help: fmt.Sprintf("User name. That can be also set by %s environment variable", userEnv)})

	passw := parser.String("p", "password", &argparse.Options{Required: false, Default: "",
		Help: fmt.Sprintf("User password. That can be also set by %s environment variable", passwEnv)})

	httpTimeout := parser.Int("t", "httpTimeout", &argparse.Options{Required: false, Default: dftHTTPTimeout,
		Help: "http timeout in seconds"})

	if err = parser.Parse(os.Args); err != nil {
		return p, fmt.Errorf(parser.Usage(err))
	}

	// check collection link. SOLR-8642
	re := regexp.MustCompile(reLink)
	strs := re.FindStringSubmatch(*cLink)
	if strs == nil {
		msg := fmt.Sprintf("wrong http link to a solr collection \"%s\"\n", *cLink)
		return nil, fmt.Errorf(msg)
	}

	// Get credentials from the environment if they aren't set in the command line
	if *user == "" {
		*user = os.Getenv(userEnv)
	}

	if *passw == "" {
		*passw = os.Getenv(passwEnv)
	}

	commit := true
	if nocommit.Happened() {
		commit = false
	}

	p = &params{
		cLink:       *cLink,
		srcDir:      *srcDir,
		nQueries:    *nQueries,
		commit:      commit,
		user:        *user,
		passw:       *passw,
		httpTimeout: *httpTimeout,
	}

	return p, nil
}

// getJSONFiles returns a slice of found json files
func getJSONFiles(dir string) (jsonFiles []string, err error) {
	fileMask := filepath.Join(dir, "*.json")
	jsonFiles, err = filepath.Glob(fileMask)
	if err != nil {
		return nil, err
	}
	return jsonFiles, nil
}

// updateFromFile does upload JSON docs from a file to a solr collection
func updateFromFile(cLink, fileName, user, passw string, params url.Values, headers map[string]string, client *http.Client) (solrResp *glsolr.Response, err error) {
	file, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	solrResp, err = glsolr.Update(cLink, user, passw, file, params, headers, client)
	if err != nil {
		return nil, err
	}

	return solrResp, nil
}

// getHeaders prepares headers
func getHeaders(agent, version string) (headers map[string]string) {
	headers = map[string]string{
		"User-Agent":      fmt.Sprintf("%s/%s (%s)", agent, version, runtime.GOOS),
		"Accept":          "application/json",
		"Connection":      "keep-alive",
		"Accept-Encoding": "gzip, deflate",
		"Content-Type":    "application/json",
	}
	return headers
}

func main() {
	p, err := parceArgs(name, reLink, os.Args)
	if err != nil {
		// In case of error print error, print an error and exit
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}

	var jFiles []string
	jFiles, err = getJSONFiles(p.srcDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(2)
	}

	njFiles := len(jFiles)
	if njFiles == 0 {
		fmt.Fprintf(os.Stderr, "No JSON files in %s\n", p.srcDir)
		os.Exit(3)
	}

	// Channel collects files to upload data from files
	var fileChan = make(chan string, njFiles)
	// Channel for info messages about done upload
	var infoChan = make(chan infoMsg)
	var wg sync.WaitGroup
	// Upload data in parallel
	for i := 0; i < p.nQueries; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				jFile, more := <-fileChan
				if !more {
					return
				}
				// Generating parameters for the update query
				params := url.Values{}
				if p.commit {
					params.Set("commit", "true")
				}

				// define headers
				headers := getHeaders(name, version)

				// Create http client
				client := &http.Client{Timeout: time.Duration(p.httpTimeout) * time.Second}

				_, err := updateFromFile(p.cLink, jFile, p.user, p.passw, params, headers, client)

				iMsg := infoMsg{
					fileName: jFile,
					err:      err,
				}
				infoChan <- iMsg
			}
		}()
	}

	// Send filename to upload data
	for _, jFile := range jFiles {
		fileChan <- jFile
	}
	close(fileChan)

	// set initial status code
	statusCode := 0
	// Output of errors and info messages
	var cnt = 0
	done := make(chan bool)
	go func() {
		for {
			iMsg, more := <-infoChan
			if !more {
				done <- true
				return
			}
			if iMsg.err != nil {
				fmt.Fprintf(os.Stderr, "%s: %s\n", iMsg.fileName, iMsg.err)
				statusCode = 20
				continue
			}
			cnt++
			fmt.Printf("%s is uploaded (%d/%d)\n", iMsg.fileName, cnt, njFiles)
		}
	}()
	wg.Wait()
	close(infoChan)
	<-done
	os.Exit(statusCode)

}
