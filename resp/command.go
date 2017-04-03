package resp

import (
	"context"
	"strconv"
)

// CommandArgument is an argument of a command
type CommandArgument []byte

// Bytes returns the argument as bytes
func (c CommandArgument) Bytes() []byte { return c }

// String returns the argument converted to a string
func (c CommandArgument) String() string { return string(c) }

// Float returns the argument as a float64.
func (c CommandArgument) Float() (float64, error) {
	return strconv.ParseFloat(string(c), 64)
}

// Int returns the argument as an int64.
func (c CommandArgument) Int() (int64, error) {
	return strconv.ParseInt(string(c), 10, 64)
}

// --------------------------------------------------------------------

// Command instances are read by a RequestReader
type Command struct {
	// Name refers to the command name
	Name string

	argc int
	argv []CommandArgument

	ctx context.Context
}

// ArgN returns the number of command arguments
func (c *Command) ArgN() int {
	return c.argc
}

// Arg returns the command argument at position i
func (c *Command) Arg(i int) CommandArgument {
	if i > -1 && i < c.argc {
		return c.Args()[i]
	}
	return nil
}

// Args returns all command argument values
func (c *Command) Args() []CommandArgument {
	return c.argv
}

// Context returns the context
func (c *Command) Context() context.Context {
	if c.ctx != nil {
		return c.ctx
	}
	return context.Background()
}

// SetContext sets the request context.
func (c *Command) SetContext(ctx context.Context) {
	if ctx != nil {
		c.ctx = ctx
	}
}

func (c *Command) reset() {
	argv := c.argv
	for i, v := range argv {
		argv[i] = v[:0]
	}
	*c = Command{argv: argv[:0]}
}

func (c *Command) grow(n int) {
	if d := n - cap(c.argv); d > 0 {
		c.argv = c.argv[:cap(c.argv)]
		c.argv = append(c.argv, make([]CommandArgument, d)...)
	} else {
		c.argv = c.argv[:n]
	}
}

// --------------------------------------------------------------------

// CommandStream instances are commands where the arguments are not
// automatically parsed but can be consumed as a stream.
type CommandStream struct {
	// Name refers to the command name
	Name string

	argc int
	ctx  context.Context

	r *bufioR
}

// ArgN returns the number of command arguments
func (c *CommandStream) ArgN() int {
	return c.argc
}

// Context returns the context
func (c *CommandStream) Context() context.Context {
	if c.ctx != nil {
		return c.ctx
	}
	return context.Background()
}

// SetContext sets the request context.
func (c *CommandStream) SetContext(ctx context.Context) {
	if ctx != nil {
		c.ctx = ctx
	}
}
