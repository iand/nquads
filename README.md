nquads - a basic nquads parser in Go

[![Build Status](https://travis-ci.org/iand/nquads.svg?branch=master)](https://travis-ci.org/iand/nquads)
[![Go Report Card](https://goreportcard.com/badge/github.com/iand/nquads)](https://goreportcard.com/report/github.com/iand/nquads)
[![go.dev reference](https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white)](https://pkg.go.dev/github.com/iand/nquads)

## Getting Started

Example of parsing an nquads file and printing out every 5000th quad

	package main

	import (
		"fmt"
		"os"
		"github.com/iand/nquads"
	)	

	func main() {
		nqfile, err := os.Open("myquads.nt")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %s", err.Error())
			os.Exit(1)
		}
		defer nqfile.Close()


		count := 0
		r := nquads.NewReader(nqfile)
		
		for r.Next()
			count++
			if count % 5000 == 0{
				fmt.Printf("%s\n", r.Quad())
			}
			
		}

		if r.Err() != nil {
			fmt.Printf("Unexpected error encountered: %v\n", r.Err())
		}

	}


## Author

* [Ian Davis](http://github.com/iand) - <http://iandavis.com/>

# License

This is free and unencumbered software released into the public domain. For more
information, see <http://unlicense.org/> or the accompanying [`UNLICENSE`](UNLICENSE) file.

## Credits

The design and logic is hugely inspired by Go's standard csv parsing library
