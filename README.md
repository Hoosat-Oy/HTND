
Hoosat Network Daemon
====

[![ISC License](http://img.shields.io/badge/license-ISC-blue.svg)](https://choosealicense.com/licenses/isc/)
[![GoDoc](https://img.shields.io/badge/godoc-reference-blue.svg)](http://godoc.org/github.com/hoosatnet/htnd)

Hoosat Network Daemon is the reference full node Hoosat Network implementation written in Go (golang).

## What is Hoosat Network

Hoosat Network is an attempt at a ASIC resistant proof-of-work cryptocurrency with instant confirmations and sub-second block times. It is fork of Hoosat and based on the [the PHANTOM protocol](https://eprint.iacr.org/2018/104.pdf), a generalization of Nakamoto consensus. 

## Requirements

Go 1.18 or later.

## Installation

#### Build from Source

- Install Go according to the installation instructions here:
  http://golang.org/doc/install

- Ensure Go was installed properly and is a supported version:

```bash
$ go version
```

- Run the following commands to obtain and install htnd including all dependencies:

```bash
$ git clone https://github.com/Hoosat-Oy/HTND
$ cd HTND
$ go install . ./cmd/...
```

- HTND (and utilities) should now be installed in `$(go env GOPATH)/bin`. If you did
  not already add the bin directory to your system path during Go installation,
  you are encouraged to do so now.


## Getting Started

HTND has several configuration options available to tweak how it runs, but all
of the basic operations work with zero configuration.

```bash
$ htnd
```

## Discord
Join our discord server using the following link: 

## Issue Tracker

The [integrated github issue tracker](https://github.com/Hoosat-Oy/HTND/issues)
is used for this project.


## Documentation

The [documentation](https://github.com//Hoosat-Oy/docs) is a work-in-progress

## License

HTND is licensed under the copyfree [ISC License](https://choosealicense.com/licenses/isc/).
