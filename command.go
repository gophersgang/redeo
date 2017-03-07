package redeo

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
)

var commandPool = sync.Pool{
	New: func() interface{} { return new(Command) },
}

// Command contains a command, arguments, and client information
type Command struct {
	Name string
	Args []string

	ctx    context.Context
	client *Client
}

func newCommand(name string, client *Client) *Command {
	cmd := commandPool.Get().(*Command)
	cmd.Name = strings.ToLower(name)
	cmd.Args = cmd.Args[:0]
	cmd.ctx = nil
	cmd.client = client
	return cmd
}

// readCommand parses a new request from a buffered connection
func readCommand(rd *bufio.Reader, c *Client) (*Command, error) {
	line, err := rd.ReadBytes('\n')
	if err != nil || len(line) < 3 {
		return nil, io.EOF
	}

	// Truncate CRLF
	line = line[:len(line)-2]

	// Return if inline
	if line[0] != '*' {
		return newCommand(string(line), c), nil
	}

	argc, ok := readSize(line[1:])
	if !ok {
		return nil, errInvalidMultiBulkLength
	}

	name, err := readArgument(rd, c)
	if err != nil {
		return nil, err
	}

	cmd := newCommand(name, c)
	for i := 1; i < argc; i++ {
		val, err := readArgument(rd, c)
		if err != nil {
			cmd.release()
			return nil, err
		}
		cmd.Args = append(cmd.Args, val)
	}
	return cmd, nil
}

// Client returns the client, may return nil
func (r *Command) Client() *Client {
	return r.client
}

// Context returns the context
func (r *Command) Context() context.Context {
	if r.ctx != nil {
		return r.ctx
	}
	return context.Background()
}

// SetContext sets the request context.
func (r *Command) SetContext(ctx context.Context) {
	if ctx != nil {
		r.ctx = ctx
	}
}

func (r *Command) release() {
	commandPool.Put(r)
}

func readSize(p []byte) (int, bool) {
	var sz int
	for _, b := range p {
		if b < '0' || b > '9' {
			return -1, false
		}
		sz = sz*10 + int(b-'0')
	}
	return sz, true
}

func readArgument(rd *bufio.Reader, c *Client) (string, error) {
	line, err := rd.ReadBytes('\n')
	if err != nil || len(line) < 3 {
		return "", io.EOF
	} else if line[0] != '$' {
		return "", fmt.Errorf("Protocol error: expected '$', got '%s'", string(line[0]))
	}

	sz, ok := readSize(line[1 : len(line)-2])
	if !ok {
		return "", errInvalidBulkLength
	}

	if mx := sz + 2; mx > cap(c.buf) {
		c.buf = make([]byte, mx)
	} else {
		c.buf = c.buf[:mx]
	}
	if _, err := io.ReadFull(rd, c.buf); err != nil {
		return "", io.EOF
	}
	return string(c.buf[:sz]), nil
}
