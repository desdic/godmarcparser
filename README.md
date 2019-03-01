# DMARC report parser

[![Build Status](https://travis-ci.org/desdic/godmarcparser.svg?branch=master)](https://travis-ci.org/desdic/godmarcparser)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

This project is very much based on [techsneezes dmarcts-report-parser](https://github.com/techsneeze/dmarcts-report-parser) but with a few modifications.

## Configuration

```json
{
  "http": {
    "port": ":8081",
          "WriteTimeout": 15,
          "ReadTimeout": 15,
          "IdleTimeout": 60
  },
  "storage": {
    "type": "postgresql",
    "url": "postgres://dmarcuser:secret@127.0.0.1:5432/dmarc?sslmode=disable"
  },
  "log": {
    "level": "info"
  },
  "directory": {
    "path": "/dmarcfiles",
    "interval": 30
  }
}
```

## Building from source

The code should work fine using go 1.11 or higher

## Bugs/Patches

If you do find any bugs please report them via the issue tracker or feel free to make a pull request.

Please make pull/feature request by:

* Fork the repo
* Create your feature branch (git checkout -b my-new-feature)
* Commit your changes (git commit -am 'Added some feature')
* Push to the branch (git push origin my-new-feature)
* Create new Pull Request

## TODO

* DNS lookup
* Integration test
* Quantine of parsed files with errors
* Deletion of successfully parsed files
* IMAP support
