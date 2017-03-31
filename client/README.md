# Client

This package implements a minimalist client for working with redis servers.

## Example

```go
package main

import (
  "fmt"
  "strings"

  "github.com/bsm/redeo/client"
)

func main() {
	client, _ := client.New(&pool.Options{
		InitialSize: 1,
	}, nil)
	defer client.Close()

	cn, _ := client.Get()
	defer client.Put(cn)

	// Build pipeline
	cn.WriteCmdString("PING")
	cn.WriteCmdString("ECHO", "HEllO")
	cn.WriteCmdString("GET", "key")
	cn.WriteCmdString("SET", "key", "value")
	cn.WriteCmdString("DEL", "key")

	// Flush pipeline to socket
	if err := cn.Flush(); err != nil {
		panic(err)
	}

	// Consume responses
	for i := 0; i < 5; i++ {
		t, err := cn.PeekType()
		if err != nil {
			return
		}

		switch t {
		case resp.TypeStatus:
			s, _ := cn.ReadStatus()
			fmt.Println(s)
		case resp.TypeString:
			s, _ := cn.ReadString()
			fmt.Println(s)
		case resp.TypeInt:
			n, _ := cn.ReadInt()
			fmt.Println(n)
		case resp.TypeNil:
			_ = cn.ReadNil()
			fmt.Println(nil)
		default:
			panic("unexpected response type")
		}
	}

}
```
