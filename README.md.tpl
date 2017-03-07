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

func main() {{ "ExampleServer" | code }}
```

More complex handlers:

```go
func main() {{ "ExampleHandlerFunc" | code }}
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

