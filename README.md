# nquads 

A basic nquads parser in Go

[![Check Status](https://github.com/iand/nquads/actions/workflows/check.yml/badge.svg)](https://github.com/iand/nquads/actions/workflows/check.yml)
[![Test Status](https://github.com/iand/nquads/actions/workflows/test.yml/badge.svg)](https://github.com/iand/nquads/actions/workflows/test.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/iand/nquads)](https://goreportcard.com/report/github.com/iand/nquads)
[![go.dev reference](https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white)](https://pkg.go.dev/github.com/iand/nquads)

## Overview

[N-Quads](https://www.w3.org/TR/n-quads/) is a serialisation format for [RDF datasets](https://www.w3.org/TR/rdf11-concepts/#section-dataset).
A dataset consists of a default graph with no name and zero or more named graphs where a graph is a composed of a set of triples. The default 
graph may be empty. 

An N-Quads file is a line-oriented format where each triple or quad statement is terminated by a period `.`.

 - IRIs are enclosed by `<` and `>`
 - Literals have a lexical value enclosed by `"` followed by an optional language tag using `@` as a delimiter, or a data type IRI using `^^` as a delimiter
 - Blank nodes have a lexical label prefixed by `_:` and the same label denotes the same blank node throughout the file.

A triple in a named graph may be written as a statement using four terms, the last of which is the name of the graph:

```
<http://example/s> <http://example/p> <http://example/o> <http://example/g> .
```

A triple in the default graph omits the fourth term:

```
<http://example/s> <http://example/p> <http://example/o> .
```

## Getting Started

Example of parsing an nquads file and printing out every 5000th quad

```Go
	package main

	import (
		"fmt"
		"os"
		"github.com/iand/nquads"
	)	

	func main() {
		nqfile, err := os.Open("myquads.nq")
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
```

## Author

* [Ian Davis](http://github.com/iand) - <http://iandavis.com/>

# License

This is free and unencumbered software released into the public domain. For more
information, see <http://unlicense.org/> or the accompanying [`UNLICENSE`](UNLICENSE) file.

## Credits

The design and logic is inspired by Go's standard csv parsing library
