# Redeo

[![GoDoc](https://godoc.org/github.com/bsm/redeo?status.svg)](https://godoc.org/github.com/bsm/redeo)
[![Build Status](https://travis-ci.org/bsm/redeo.png?branch=master)](https://travis-ci.org/bsm/redeo)
[![Go Report Card](https://goreportcard.com/badge/github.com/bsm/redeo)](https://goreportcard.com/report/github.com/bsm/redeo)

High-performance framework for building redis-protocol compatible TCP
servers/services. Optimised for speed!

## Full Documentation

For documentation and examples, please see https://godoc.org/github.com/bsm/redeo.

## Examples

A simple example with two commands:

```go
package main

import (
  "net"

  "github.com/bsm/redeo"
)

func main() {

	srv := redeo.NewServer(nil)
	srv.HandleFunc("ping", func(w *redeo.ResponseBuffer, _ *redeo.Command) {
		w.AppendInlineString("PONG")
	})
	srv.HandleFunc("info", func(w *redeo.ResponseBuffer, _ *redeo.Command) {
		w.AppendString(srv.Info().String())
	})

	lis, err := net.Listen("tcp", ":9736")
	if err != nil {
		panic(err)
	}
	defer lis.Close()

	srv.Serve(lis)
}
```

More complex handlers:

```go
func main() {
	mu := sync.RWMutex{}
	myData := make(map[string]map[string]string)
	srv := redeo.NewServer(nil)

	srv.HandleFunc("hset", func(w *redeo.ResponseBuffer, c *redeo.Command) {

		if len(c.Args) != 3 {
			w.AppendError(redeo.WrongNumberOfArgs(c.Name))
			return
		}

		mu.Lock()
		defer mu.Unlock()

		key, ok := myData[c.Args[0]]
		if !ok {
			key = make(map[string]string)
			myData[c.Args[0]] = key
		}

		_, ok = key[c.Args[1]]

		key[c.Args[1]] = c.Args[2]

		if ok {
			w.AppendInt(0)
		} else {
			w.AppendInt(1)
		}
	})

	srv.HandleFunc("hget", func(w *redeo.ResponseBuffer, c *redeo.Command) {
		if len(c.Args) != 2 {
			w.AppendError(redeo.WrongNumberOfArgs(c.Name))
			return
		}

		mu.RLock()
		defer mu.RUnlock()

		key, ok := myData[c.Args[0]]
		if !ok {
			w.AppendNil()
			return
		}

		val, ok := key[c.Args[1]]
		if !ok {
			w.AppendNil()
			return
		}

		w.AppendString(val)
	})
}
```

## Licence

```
Copyright 2017 Black Square Media Ltd

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
```

