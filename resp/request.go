package resp

import (
	"io"
)

// RequestReader is used by servers to wrap a client connection and convert
// requests into commands.
type RequestReader struct {
	r *bufioR
}

// NewRequestReader wraps any reader interface
func NewRequestReader(rd io.Reader) *RequestReader {
	r := new(bufioR)
	r.reset(mkbuf(), rd)
	return &RequestReader{r: r}
}

// Buffered returns the number of unread bytes.
func (r *RequestReader) Buffered() int {
	return r.r.Buffered()
}

// Reset resets the reader to a new reader and recycles internal buffers.
func (r *RequestReader) Reset(rd io.Reader) {
	r.r.Reset(rd)
}

// ReadCmd reads the next command. It optionally recycles the cmd passed.
func (r *RequestReader) ReadCmd(cmd *Command) (*Command, error) {
	if cmd != nil {
		cmd.reset()
	}

	c, err := r.r.PeekByte()
	if err != nil {
		return cmd, err
	}

	if c == '*' {
		return r.readMultiBulkCmd(cmd)
	}
	return r.readInlineCmd(cmd)
}

func (r *RequestReader) readInlineCmd(cmd *Command) (*Command, error) {
	if cmd == nil {
		cmd = new(Command)
	}

	var name []byte
	defer func() {
		if name != nil {
			cmd.Name = string(name)
		}
	}()

	inField := false
	for {
		b, err := r.r.ReadByte()
		if err != nil {
			return cmd, err
		}

		switch b {
		case '\r', '\n':
			r.r.DiscardCRLF()
			if name == nil {
				return r.ReadCmd(cmd)
			}
			return cmd, nil
		case ' ', '\t':
			inField = false
		default:
			if !inField && name != nil {
				cmd.argc++
				cmd.grow(cmd.argc)
			}
			if pos := cmd.argc - 1; pos > -1 {
				cmd.argv[pos] = append(cmd.argv[pos], b)
			} else {
				name = append(name, b)
			}
			inField = true
		}
	}
}

func (r *RequestReader) readMultiBulkCmd(cmd *Command) (*Command, error) {
	sz, err := r.r.ReadArrayLen()
	if err != nil {
		return cmd, err
	}
	if sz < 1 {
		return r.ReadCmd(cmd)
	}

	if cmd == nil {
		cmd = new(Command)
	}
	cmd.argc = sz - 1
	cmd.grow(cmd.argc)

	cmd.Name, err = r.r.ReadString()
	if err != nil {
		return cmd, err
	}

	for i := 0; i < cmd.argc; i++ {
		bb, err := r.r.ReadBytes()
		if err != nil {
			return cmd, err
		}
		cmd.argv[i] = append(cmd.argv[i], bb...)
	}

	return cmd, err
}

// --------------------------------------------------------------------

// RequestWriter is used by clients to send commands to servers.
type RequestWriter struct {
	w *bufioW
}

// NewRequestWriter wraps any Writer interface
func NewRequestWriter(wr io.Writer) *RequestWriter {
	w := new(bufioW)
	w.reset(mkbuf(), wr)
	return &RequestWriter{w: w}
}

// Reset resets the writer with an new interface
func (w *RequestWriter) Reset(wr io.Writer) {
	w.w.Reset(wr)
}

// Buffered returns the number of buffered bytes
func (w *RequestWriter) Buffered() int {
	return w.w.Buffered()
}

// Flush flushes the output buffer. Call this after you have completed your pipeline
func (w *RequestWriter) Flush() error {
	return w.w.Flush()
}

// WriteCmd writes a full command as part of a pipeline. To execute the pipeline,
// you must call Flush.
func (w *RequestWriter) WriteCmd(cmd string, args ...[]byte) {
	w.w.AppendArrayLen(len(args) + 1)
	w.w.AppendString(cmd)
	for _, arg := range args {
		w.w.AppendBytes(arg)
	}
}

// WriteCmdString writes a full command as part of a pipeline. To execute the pipeline,
// you must call Flush.
func (w *RequestWriter) WriteCmdString(cmd string, args ...string) {
	w.w.AppendArrayLen(len(args) + 1)
	w.w.AppendString(cmd)
	for _, arg := range args {
		w.w.AppendString(arg)
	}
}

// WriteMultiBulkSize is a low-level method to write a multibulk size.
// For normal operation, use WriteCmd or WriteCmdString.
func (w *RequestWriter) WriteMultiBulkSize(n int) error {
	if n < 0 {
		return errInvalidMultiBulkLength
	}
	w.w.AppendArrayLen(n)
	return nil
}

// WriteBulk is a low-level method to write a bulk.
// For normal operation, use WriteCmd or WriteCmdString.
func (w *RequestWriter) WriteBulk(b []byte) {
	w.w.AppendBytes(b)
}

// WriteBulkString is a low-level method to write a bulk.
// For normal operation, use WriteCmd or WriteCmdString.
func (w *RequestWriter) WriteBulkString(s string) {
	w.w.AppendString(s)
}

// WriteFromN is a low-level method to copy a large bulk of data directly to the writer.
// For normal operation, use WriteCmd or WriteCmdString.
func (w *RequestWriter) WriteFromN(r io.Reader, n int) error {
	return w.w.WriteFromN(r, n)
}
