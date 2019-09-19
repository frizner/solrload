# solrload
[![Go Report Card](https://goreportcard.com/badge/github.com/frizner/solrload)](https://goreportcard.com/report/github.com/frizner/solrload)

`solrload` is the utility to upload JSON documents into a Solr collection using update queries in parallel:
```sh
$ SOLRUSER="solruser" SOLRPASSW="solrpassword" solrload -c "http://solrsrv02:8983/solr/gettingstarted" -s "solrsrv01.8983.gettingstarted.20181029-163603"
solrsrv01.8983.gettingstarted.20181029-163603/solrsrv01.8983.gettingstarted.5.json is uploaded (1/5)
solrsrv01.8983.gettingstarted.20181029-163603/solrsrv01.8983.gettingstarted.3.json is uploaded (2/5)
solrsrv01.8983.gettingstarted.20181029-163603/solrsrv01.8983.gettingstarted.1.json is uploaded (3/5)
solrsrv01.8983.gettingstarted.20181029-163603/solrsrv01.8983.gettingstarted.2.json is uploaded (4/5)
solrsrv01.8983.gettingstarted.20181029-163603/solrsrv01.8983.gettingstarted.4.json is uploaded (5/5)
```

## Features
- Uploading documents into a Solr collection in parallel. Even if a collection has only one shard, uploading in parallel can sufficiently decrees time of indexing.
- `solrload` can be used in a tandem with [solrdump](https://github.com/frizner/solrdump) utility.
```sh
SOLRUSER="solruser1" SOLRPASSW="solrpassword1" solrdump -c "http://solrsrv01:8983/solr/gettingstarted" -r 50000 -s "id asc"
$ ls
solrsrv01.8983.gettingstarted.20181017-160227
$ SOLRUSER="solruser2" SOLRPASSW="solrpassword2" solrload -c "http://solrsrv02:8983/solr/gettingstarted" -s "solrsrv01.8983.gettingstarted.20181017-160227"
solrsrv01.8983.gettingstarted.20181017-160227/solrsrv01.8983.gettingstarted.2.json is uploaded (1/523)
solrsrv01.8983.gettingstarted.20181017-160227/solrsrv01.8983.gettingstarted.1.json is uploaded (2/523)
...
solrsrv01.8983.gettingstarted.20181017-160227/solrsrv01.8983.gettingstarted.523.json is uploaded (523/523)
```

## Installation
### Binaries
Download the binary from the [releases](https://github.com/frizner/solrload/releases) page.
### From Source
You can use the `go` tool to install `solrload`:
```sh
$ go get "github.com/frizner/solrload"
$ go install "github.com/frizner/solrload/cmd/solrload"
```
This installs the command into the bin sub-folder of wherever your $GOPATH environment variable points. If this directory is already in your $PATH, then you should be good to go.

If you have already pulled down this repo to a location that is not in your $GOPATH and want to build from the sources, you can cd into the repo and then run make install.

## Usage
```sh
$ solrload -h
usage: solrload <Command> [-h|--help] -c|--collink "<value>" [-n|--nqueries
                <integer>] [-s|--src "<value>"] [-u|--user "<value>"]
                [-p|--password "<value>"] [-t|--httpTimeout <integer>]

                solrload uploads documents from JSON files in a Solr collection
                (index) using the update queries in parallel 

Commands:

  --nocommit  Won't do the commit after the each update query

Arguments:

  -h  --help         Print help information
  -c  --collink      http link to a Solr collection like
                     http[s]://address[:port]/solr/collection
  -n  --nqueries     Number of updating queries in parallel. Default: 8
  -s  --src          Path to the dump directory with JSON files to upload.
                     Default: .
  -u  --user         User name. That can be also set by SOLRUSER environment
                     variable. Default: 
  -p  --password     User password. That can be also set by SOLRPASSW
                     environment variable. Default: 
  -t  --httpTimeout  http timeout in seconds. Default: 180
```

### License
`solrload` is released under the MIT License. See [LICENSE](https://github.com/frizner/solrload/blob/master/LICENSE).
