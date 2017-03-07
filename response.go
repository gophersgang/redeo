package redeo

import (
	"io"
	"strconv"
	"sync"
)

var (
	binCRLF = []byte("\r\n")
	binOK   = []byte("+OK\r\n")
	binZERO = []byte(":0\r\n")
	binONE  = []byte(":1\r\n")
	binNIL  = []byte("$-1\r\n")
)

var copyBufPool = sync.Pool{New: func() interface{} { return make([]byte, 32*1024) }}

const maxResponseBufferSize = 16 * 1024

// ResponseBuffer is a wrapper around a connection which helps
// writing correctly formatted responses to clients.
type ResponseBuffer struct {
	w io.Writer
	b []byte

	dirty bool
}

// NewResponseBuffer returns a brand-new responder.
// Can be used with a bytes.Buffer for testing
func NewResponseBuffer(w io.Writer) *ResponseBuffer {
	return &ResponseBuffer{w: w}
}

// AppendArrayLen appends an array header to the output buffer
func (w *ResponseBuffer) AppendArrayLen(n int) {
	w.b = append(w.b, '*')
	w.b = append(w.b, strconv.Itoa(n)...)
	w.b = append(w.b, binCRLF...)
}

// AppendBytes appends bulk bytes to the output buffer
func (w *ResponseBuffer) AppendBytes(p []byte) {
	w.b = append(w.b, '$')
	w.b = append(w.b, strconv.Itoa(len(p))...)
	w.b = append(w.b, binCRLF...)
	w.b = append(w.b, p...)
	w.b = append(w.b, binCRLF...)
}

// AppendString appends a bulk string to the output buffer
func (w *ResponseBuffer) AppendString(s string) {
	w.b = append(w.b, '$')
	w.b = append(w.b, strconv.Itoa(len(s))...)
	w.b = append(w.b, binCRLF...)
	w.b = append(w.b, s...)
	w.b = append(w.b, binCRLF...)
}

// AppendInlineBytes appends inline bytes to the output buffer
func (w *ResponseBuffer) AppendInlineBytes(p []byte) {
	w.b = append(w.b, '+')
	w.b = append(w.b, p...)
	w.b = append(w.b, binCRLF...)
}

// AppendInlineString appends an inline string to the output buffer
func (w *ResponseBuffer) AppendInlineString(s string) {
	w.b = append(w.b, '+')
	w.b = append(w.b, s...)
	w.b = append(w.b, binCRLF...)
}

// AppendError appends an error message to the output buffer
func (w *ResponseBuffer) AppendError(msg string) {
	w.b = append(w.b, '-')
	w.b = append(w.b, msg...)
	w.b = append(w.b, binCRLF...)
}

// AppendInt appends a numeric response to the output buffer
func (w *ResponseBuffer) AppendInt(n int64) {
	switch n {
	case 0:
		w.b = append(w.b, binZERO...)
	case 1:
		w.b = append(w.b, binONE...)
	default:
		w.b = append(w.b, ':')
		w.b = append(w.b, strconv.FormatInt(n, 10)...)
		w.b = append(w.b, binCRLF...)
	}
}

// AppendNil appends a nil-value to the output buffer
func (w *ResponseBuffer) AppendNil() {
	w.b = append(w.b, binNIL...)
}

// AppendOK appends "OK" to the output buffer
func (w *ResponseBuffer) AppendOK() {
	w.b = append(w.b, binOK...)
}

// CopyN flushes the existing buffer and copies n bytes from the reader directly to
// the client connection.
func (w *ResponseBuffer) CopyN(src io.Reader, n int64) error {
	w.b = append(w.b, '$')
	w.b = append(w.b, strconv.FormatInt(n, 10)...)
	w.b = append(w.b, binCRLF...)
	if err := w.Flush(); err != nil {
		return err
	}

	buf := copyBufPool.Get().([]byte)
	defer copyBufPool.Put(buf)

	_, err := io.CopyBuffer(w.w, io.LimitReader(src, n), buf)
	if err != nil {
		return err
	}

	w.b = append(w.b, binCRLF...)
	return nil
}

// Buffered returns the number of pending bytes
func (w *ResponseBuffer) Buffered() int {
	return len(w.b)
}

// Flush flushes pending buffer
func (w *ResponseBuffer) Flush() error {
	if len(w.b) == 0 {
		return nil
	}

	if _, err := w.w.Write(w.b); err != nil {
		return err
	}

	w.b = w.b[:0]
	w.dirty = true
	return nil
}

func (w *ResponseBuffer) reset(wr io.Writer) {
	w.w = wr
	w.b = w.b[:0]
	w.dirty = false
}
