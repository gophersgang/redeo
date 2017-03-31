# Redeo

[![GoDoc](https://godoc.org/github.com/bsm/redeo?status.svg)](https://godoc.org/github.com/bsm/redeo)
[![Build Status](https://travis-ci.org/bsm/redeo.png?branch=master)](https://travis-ci.org/bsm/redeo)
[![Go Report Card](https://goreportcard.com/badge/github.com/bsm/redeo)](https://goreportcard.com/report/github.com/bsm/redeo)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

The high-performance Swiss Army Knife for building redis-protocol compatible servers/services.

## Parts

This repository is organised into multiple components:

* [root](./) package contains the framework for building redis-protocol compatible,
  high-performance servers.
* [resp](./resp/) implements low-level primitives for dealing with
  RESP (REdis Serialization Protocol), client and server-side. It
  contains basic wrappers for readers and writers to read/write requests and
  responses.
* [client](./client/) contains a minimalist pooled client.

For full documentation and examples, please see the individual packages and the
official API documentation: https://godoc.org/github.com/bsm/redeo.

## Examples

A simple server example with two commands:

```go
package main

import (
  "net"

  "github.com/bsm/redeo"
)

func main() {
	// Init server and define handlers
	srv := redeo.NewServer(nil)
	srv.HandleFunc("ping", func(w resp.ResponseWriter, _ *resp.Command) {
		w.AppendInlineString("PONG")
	})
	srv.HandleFunc("info", func(w resp.ResponseWriter, _ *resp.Command) {
		w.AppendString(srv.Info().String())
	})

	// Open a new listener
	lis, err := net.Listen("tcp", ":9736")
	if err != nil {
		panic(err)
	}
	defer lis.Close()

	// Start serving (blocking)
	srv.Serve(lis)
}
```

More complex handlers:

```go
func main() {
	mu := sync.RWMutex{}
	myData := make(map[string]map[string]string)
	srv := redeo.NewServer(nil)

	// handle HSET
	srv.HandleFunc("hset", func(w resp.ResponseWriter, c *resp.Command) {
		// validate arguments
		if c.ArgN() != 3 {
			w.AppendError(redeo.WrongNumberOfArgs(c.Name))
			return
		}

		// lock for write
		mu.Lock()
		defer mu.Unlock()

		// fetch (find-or-create) key
		hash, ok := myData[c.Arg(0).String()]
		if !ok {
			hash = make(map[string]string)
			myData[c.Arg(0).String()] = hash
		}

		// check if field already exists
		_, ok = hash[c.Arg(1).String()]

		// set field
		hash[c.Arg(1).String()] = c.Arg(2).String()

		// respond
		if ok {
			w.AppendInt(0)
		} else {
			w.AppendInt(1)
		}
	})

	// handle HGET
	srv.HandleFunc("hget", func(w resp.ResponseWriter, c *resp.Command) {
		if c.ArgN() != 2 {
			w.AppendError(redeo.WrongNumberOfArgs(c.Name))
			return
		}

		mu.RLock()
		defer mu.RUnlock()

		hash, ok := myData[c.Arg(0).String()]
		if !ok {
			w.AppendNil()
			return
		}

		val, ok := hash[c.Arg(1).String()]
		if !ok {
			w.AppendNil()
			return
		}

		w.AppendString(val)
	})
}
```
