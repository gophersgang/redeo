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

// Command instances are parsed by a RequestReader
type Command struct {
	// Name refers to the command name
	Name string

	baseCmd
}

// Arg returns the command argument at position i
func (c *Command) Arg(i int) CommandArgument {
	if i > -1 && i < c.argc {
		return c.Args()[i]
	}
	return nil
}

// Args returns all command argument values
func (c *Command) Args() []CommandArgument { return c.argv }

func (c *Command) reset() {
	c.baseCmd.reset()
	*c = Command{baseCmd: c.baseCmd}
}

func (c *Command) parse(r *bufioR) error {
	x, err := r.PeekByte()
	if err != nil {
		return err
	}

	if x == '*' {
		err = c.parseMultiBulk(r)
	} else {
		err = c.parseInline(r)
	}
	if err != nil {
		return err
	}

	c.Name = string(c.name)
	return nil
}

func (c *Command) parseMultiBulk(r *bufioR) error {
	n, err := r.ReadArrayLen()
	if err != nil {
		return err
	}
	if n < 1 {
		return c.parse(r)
	}

	c.argc = n - 1
	c.grow(c.argc)

	c.name, err = r.ReadBulk(c.name)
	if err != nil {
		return err
	}

	for i := 0; i < c.argc; i++ {
		c.argv[i], err = r.ReadBulk(c.argv[i])
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *Command) parseInline(r *bufioR) error {
	line, err := r.ReadLine()
	if err != nil {
		return err
	}

	hasName := false
	inWord := false
	for _, x := range line.Trim() {
		switch x {
		case ' ', '\t':
			inWord = false
		default:
			if !inWord && hasName {
				c.argc++
				c.grow(c.argc)
			}
			if pos := c.argc - 1; pos > -1 {
				c.argv[pos] = append(c.argv[pos], x)
			} else {
				hasName = true
				c.name = append(c.name, x)
			}
			inWord = true
		}
	}
	if !hasName {
		return c.parse(r)
	}
	return nil
}

// --------------------------------------------------------------------

// CommandStream instances are created by a RequestReader
type CommandStream struct {
	// Name refers to the command name
	Name string

	baseCmd
}

// --------------------------------------------------------------------

type baseCmd struct {
	argc int
	argv []CommandArgument
	name []byte

	ctx context.Context
}

// ArgN returns the number of command arguments
func (c *baseCmd) ArgN() int {
	return c.argc
}

// Context returns the context
func (c *baseCmd) Context() context.Context {
	if c.ctx != nil {
		return c.ctx
	}
	return context.Background()
}

// SetContext sets the request context.
func (c *baseCmd) SetContext(ctx context.Context) {
	if ctx != nil {
		c.ctx = ctx
	}
}

func (c *baseCmd) grow(n int) {
	if d := n - cap(c.argv); d > 0 {
		c.argv = c.argv[:cap(c.argv)]
		c.argv = append(c.argv, make([]CommandArgument, d)...)
	} else {
		c.argv = c.argv[:n]
	}
}

func (c *baseCmd) reset() {
	argv := c.argv
	for i, v := range argv {
		argv[i] = v[:0]
	}
	*c = baseCmd{
		argv: argv[:0],
		name: c.name[:0],
	}
}
