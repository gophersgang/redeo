package redeo_test

import (
	"bytes"
	"fmt"
	"net"
	"strings"
	"sync"

	"github.com/bsm/redeo"
)

func ExampleServer() {
	// Init server and define handlers
	srv := redeo.NewServer(nil)
	srv.HandleFunc("ping", func(w *redeo.ResponseBuffer, _ *redeo.Command) {
		w.AppendInlineString("PONG")
	})
	srv.HandleFunc("info", func(w *redeo.ResponseBuffer, _ *redeo.Command) {
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

func ExampleHandlerFunc() {
	mu := sync.RWMutex{}
	myData := make(map[string]map[string]string)
	srv := redeo.NewServer(nil)

	// handle HSET
	srv.HandleFunc("hset", func(w *redeo.ResponseBuffer, c *redeo.Command) {
		// validate arguments
		if len(c.Args) != 3 {
			w.AppendError(redeo.WrongNumberOfArgs(c.Name))
			return
		}

		// lock for write
		mu.Lock()
		defer mu.Unlock()

		// fetch (find-or-create) key
		key, ok := myData[c.Args[0]]
		if !ok {
			key = make(map[string]string)
			myData[c.Args[0]] = key
		}

		// check if field already exists
		_, ok = key[c.Args[1]]

		// set field
		key[c.Args[1]] = c.Args[2]

		// respond
		if ok {
			w.AppendInt(0)
		} else {
			w.AppendInt(1)
		}
	})

	// handle HGET
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

func ExampleResponseBuffer_AppendString() {
	b := new(bytes.Buffer)
	w := redeo.NewResponseBuffer(b)

	w.AppendString("PONG")
	w.Flush()
	fmt.Printf("%q\n", b.String())

	// Output:
	// "$4\r\nPONG\r\n"
}

func ExampleResponseBuffer_AppendInlineString() {
	b := new(bytes.Buffer)
	w := redeo.NewResponseBuffer(b)

	w.AppendInlineString("PONG")
	w.Flush()
	fmt.Printf("%q\n", b.String())

	// Output:
	// "+PONG\r\n"
}

func ExampleResponseBuffer_AppendNil() {
	b := new(bytes.Buffer)
	w := redeo.NewResponseBuffer(b)

	w.AppendNil()
	w.Flush()
	fmt.Printf("%q\n", b.String())

	// Output:
	// "$-1\r\n"
}

func ExampleResponseBuffer_AppendInt() {
	b := new(bytes.Buffer)
	w := redeo.NewResponseBuffer(b)

	w.AppendInt(33)
	w.Flush()
	fmt.Printf("%q\n", b.String())

	// Output:
	// ":33\r\n"
}

func ExampleResponseBuffer_AppendArrayLen() {
	b := new(bytes.Buffer)
	w := redeo.NewResponseBuffer(b)

	w.AppendArrayLen(3)
	w.AppendString("item 1")
	w.AppendNil()
	w.AppendString("item 2")
	w.Flush()
	fmt.Printf("%q\n", b.String())

	// Output:
	// "*3\r\n$6\r\nitem 1\r\n$-1\r\n$6\r\nitem 2\r\n"
}

func ExampleResponseBuffer_CopyN() {
	b := new(bytes.Buffer)
	w := redeo.NewResponseBuffer(b)

	w.CopyN(strings.NewReader("a streamer"), 8)
	w.Flush()
	fmt.Printf("%q\n", b.String())

	// Output:
	// "$8\r\na stream\r\n"
}

func ExampleResponseBuffer_CopyN_in_array() {
	b := new(bytes.Buffer)
	w := redeo.NewResponseBuffer(b)

	w.AppendArrayLen(2)
	w.AppendString("item 1")
	w.CopyN(strings.NewReader("item 2"), 6)
	w.Flush()
	fmt.Printf("%q\n", b.String())

	// Output:
	// "*2\r\n$6\r\nitem 1\r\n$6\r\nitem 2\r\n"
}
