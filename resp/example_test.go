package resp_test

import (
	"bytes"
	"fmt"
	"net"
	"strings"

	"github.com/bsm/redeo/resp"
)

func Example_client() {
	cn, _ := net.Dial("tcp", "127.0.0.1:6379")
	defer cn.Close()

	// Wrap connection
	w := resp.NewRequestWriter(cn)
	r := resp.NewResponseReader(cn)

	// Write pipeline
	w.WriteCmdString("PING")
	w.WriteCmdString("ECHO", "HEllO")
	w.WriteCmdString("GET", "key")
	w.WriteCmdString("SET", "key", "value")
	w.WriteCmdString("DEL", "key")

	// Flush pipeline
	if err := w.Flush(); err != nil {
		panic(err)
	}

	// Consume responses
	for i := 0; i < 5; i++ {
		t, err := r.PeekType()
		if err != nil {
			return
		}

		switch t {
		case resp.TypeStatus:
			s, _ := r.ReadStatus()
			fmt.Println(s)
		case resp.TypeString:
			s, _ := r.ReadString()
			fmt.Println(s)
		case resp.TypeInt:
			n, _ := r.ReadInt()
			fmt.Println(n)
		case resp.TypeNil:
			_ = r.ReadNil()
			fmt.Println(nil)
		default:
			panic("unexpected response type")
		}
	}

	// Output:
	// PONG
	// HEllO
	// <nil>
	// OK
	// 1
}
func ExampleRequestReader() {
	cn := strings.NewReader("*1\r\n$4\r\nPING\r\n*2\r\n$4\r\nEcHO\r\n$5\r\nHeLLO\r\n")
	r := resp.NewRequestReader(cn)

	// read command
	cmd, _ := r.ReadCmd(nil)
	fmt.Println(cmd.Name)
	for i := 0; i < cmd.ArgN(); i++ {
		fmt.Println(i, cmd.Arg(i))
	}

	// read command, recycle previous instance
	cmd, _ = r.ReadCmd(cmd)
	fmt.Println(cmd.Name)
	for i := 0; i < cmd.ArgN(); i++ {
		fmt.Println(i, cmd.Arg(i))
	}

	// Output:
	// PING
	// EcHO
	// 0 HeLLO
}

func ExampleResponseWriter() {
	buf := new(bytes.Buffer)
	w := resp.NewResponseWriter(buf)

	// Append OK response
	w.AppendOK()

	// Append a number
	w.AppendInt(33)

	// Append an array
	w.AppendArrayLen(3)
	w.AppendString("Adam")
	w.AppendString("Had'em")
	w.AppendNil()

	// Writer data must be flushed manually
	fmt.Println(buf.Len(), w.Buffered())
	if err := w.Flush(); err != nil {
		panic(err)
	}

	// Once flushed, it will be sent to the underlying writer
	// as a bulk
	fmt.Println(buf.Len(), w.Buffered())
	fmt.Printf("%q\n", buf.String())

	// Output:
	// 0 41
	// 41 0
	// "+OK\r\n:33\r\n*3\r\n$4\r\nAdam\r\n$6\r\nHad'em\r\n$-1\r\n"
}

func ExampleResponseWriter_AppendString() {
	b := new(bytes.Buffer)
	w := resp.NewResponseWriter(b)

	w.AppendString("PONG")
	w.Flush()
	fmt.Printf("%q\n", b.String())

	// Output:
	// "$4\r\nPONG\r\n"
}

func ExampleResponseWriter_AppendInlineString() {
	b := new(bytes.Buffer)
	w := resp.NewResponseWriter(b)

	w.AppendInlineString("PONG")
	w.Flush()
	fmt.Printf("%q\n", b.String())

	// Output:
	// "+PONG\r\n"
}

func ExampleResponseWriter_AppendNil() {
	b := new(bytes.Buffer)
	w := resp.NewResponseWriter(b)

	w.AppendNil()
	w.Flush()
	fmt.Printf("%q\n", b.String())

	// Output:
	// "$-1\r\n"
}

func ExampleResponseWriter_AppendInt() {
	b := new(bytes.Buffer)
	w := resp.NewResponseWriter(b)

	w.AppendInt(33)
	w.Flush()
	fmt.Printf("%q\n", b.String())

	// Output:
	// ":33\r\n"
}

func ExampleResponseWriter_AppendArrayLen() {
	b := new(bytes.Buffer)
	w := resp.NewResponseWriter(b)

	w.AppendArrayLen(3)
	w.AppendString("item 1")
	w.AppendNil()
	w.AppendString("item 2")
	w.Flush()
	fmt.Printf("%q\n", b.String())

	// Output:
	// "*3\r\n$6\r\nitem 1\r\n$-1\r\n$6\r\nitem 2\r\n"
}

func ExampleResponseWriter_WriteFromN() {
	b := new(bytes.Buffer)
	w := resp.NewResponseWriter(b)

	w.WriteFromN(strings.NewReader("a streamer"), 8)
	w.Flush()
	fmt.Printf("%q\n", b.String())

	// Output:
	// "$8\r\na stream\r\n"
}

func ExampleResponseWriter_WriteFromN_in_array() {
	b := new(bytes.Buffer)
	w := resp.NewResponseWriter(b)

	w.AppendArrayLen(2)
	w.AppendString("item 1")
	w.WriteFromN(strings.NewReader("item 2"), 6)
	w.Flush()
	fmt.Printf("%q\n", b.String())

	// Output:
	// "*2\r\n$6\r\nitem 1\r\n$6\r\nitem 2\r\n"
}
