package resp

import (
	"bytes"
	"io"
	"strconv"
)

type bufioR struct {
	rd  io.Reader
	buf []byte

	r, w int
}

// Buffered returns the number of buffered bytes
func (b *bufioR) Buffered() int {
	return b.w - b.r
}

func (b *bufioR) PeekByte() (byte, error) {
	if err := b.require(1); err != nil {
		return 0, err
	}
	return b.buf[b.r], nil
}

func (b *bufioR) PeekType() (t ResponseType, err error) {
	if err = b.require(1); err != nil {
		return
	}

	switch b.buf[b.r] {
	case '*':
		t = TypeArray
	case '$':
		if err = b.require(2); err != nil {
			return
		}
		if b.buf[b.r+1] == '-' {
			t = TypeNil
		} else {
			t = TypeBulk
		}
	case '+':
		t = TypeStatus
	case '-':
		t = TypeError
	case ':':
		t = TypeInt
	}
	return
}

func (b *bufioR) ReadNil() error {
	line, err := b.ReadLine()
	if err != nil {
		return err
	}
	if len(line) < 3 || !bytes.Equal(line[:3], binNIL[:3]) {
		return errNotANilMessage
	}
	return nil
}

func (b *bufioR) ReadInt() (int64, error) {
	line, err := b.ReadLine()
	if err != nil {
		return 0, err
	}
	return line.ParseInt()
}

func (b *bufioR) ReadError() (string, error) {
	line, err := b.ReadLine()
	if err != nil {
		return "", err
	}
	return line.ParseMessage('-')
}

func (b *bufioR) ReadStatus() (string, error) {
	line, err := b.ReadLine()
	if err != nil {
		return "", err
	}
	return line.ParseMessage('+')
}

func (b *bufioR) ReadArrayLen() (int, error) {
	line, err := b.ReadLine()
	if err != nil {
		return 0, err
	}
	return line.ParseSize('*', errInvalidMultiBulkLength)
}

func (b *bufioR) ReadBulkLen() (int, error) {
	line, err := b.ReadLine()
	if err != nil {
		return 0, err
	}
	return line.ParseSize('$', errInvalidBulkLength)
}

func (b *bufioR) ReadBulk() ([]byte, error) {
	sz, err := b.ReadBulkLen()
	if err != nil {
		return nil, err
	}

	if err := b.require(sz); err != nil {
		return nil, err
	}

	bb := make([]byte, sz)
	copy(bb, b.buf[b.r:b.r+sz])

	b.r += sz
	b.skip(2)
	return bb, nil
}

func (b *bufioR) ReadBulkString() (string, error) {
	sz, err := b.ReadBulkLen()
	if err != nil {
		return "", err
	}

	if err := b.require(sz); err != nil {
		return "", err
	}

	s := string(b.buf[b.r : b.r+sz])

	b.r += sz
	b.skip(2)
	return s, nil
}

func (b *bufioR) SkipBulk() error {
	sz, err := b.ReadBulkLen()
	if err != nil {
		return err
	}

	// if bulk doesn't overflow buffer
	extra := sz - b.Buffered()
	if extra < 1 {
		b.r += sz
		b.skip(2)
		return nil
	}

	// otherwise, reset buffer ...
	b.r = 0
	b.w = 0

	// ... and discard the extra bytes
	x := extra + 2
	r := io.LimitReader(b.rd, int64(x))
	for {
		n, err := r.Read(b.buf)
		x -= n

		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}
	}
	if x != 0 {
		return io.EOF
	}
	return nil
}

func (b *bufioR) PeekN(offset, n int) ([]byte, error) {
	if err := b.require(offset + n); err != nil {
		return nil, err
	}
	return b.buf[b.r+offset : b.r+offset+n], nil
}

// PeekLine returns the next line until CRLF without reading it
func (b *bufioR) PeekLine(offset int) (bufioLn, error) {
	index := -1

	// try to find the end of the line
	start := b.r + offset
	if start < b.w {
		index = bytes.IndexByte(b.buf[start:b.w], '\n')
	}

	// try to read more data into the buffer if not in the buffer
	if index < 0 {
		if err := b.fill(); err != nil {
			return nil, err
		}
		start = b.r + offset
		if start < b.w {
			index = bytes.IndexByte(b.buf[start:b.w], '\n')
		}
	}

	// fail if still nothing found
	if index < 0 {
		return nil, nil
	}
	return bufioLn(b.buf[start : start+index+1]), nil
}

// ReadLine returns the next line until CRLF
func (b *bufioR) ReadLine() (bufioLn, error) {
	line, err := b.PeekLine(0)
	b.r += len(line)
	return line, err
}

// Reset resets the reader with an new interface
func (b *bufioR) Reset(r io.Reader) {
	b.reset(b.buf, r)
}

// require ensures that sz bytes are buffered
func (b *bufioR) require(sz int) error {
	extra := sz - b.Buffered()
	if extra < 1 {
		return nil
	}

	// compact first
	b.compact()

	// grow the buffer if necessary
	if n := b.w + extra; n > len(b.buf) {
		buf := make([]byte, n)
		copy(buf, b.buf[:b.w])
		b.buf = buf
	}

	// read data into buffer
	n, err := io.ReadAtLeast(b.rd, b.buf[b.w:], extra)
	b.w += n
	return err
}

func (b *bufioR) skip(sz int) {
	if b.Buffered() >= sz {
		b.r += sz
	}
}

// fill tries to read more data into the buffer
func (b *bufioR) fill() error {
	b.compact()

	if b.w < len(b.buf) {
		n, err := b.rd.Read(b.buf[b.w:])
		b.w += n
		return err
	}
	return nil
}

// compact moves the unread chunk to the beginning of the buffer
func (b *bufioR) compact() {
	if b.r > 0 {
		copy(b.buf, b.buf[b.r:b.w])
		b.w -= b.r
		b.r = 0
	}
}

func (b *bufioR) reset(buf []byte, rd io.Reader) {
	*b = bufioR{buf: buf, rd: rd}
}

// --------------------------------------------------------------------

type bufioLn []byte

// Trim truncates CRLF
func (ln bufioLn) Trim() bufioLn {
	n := len(ln)
	for ; n > 0; n-- {
		if c := ln[n-1]; c != '\r' && c != '\n' {
			break
		}
	}
	return ln[:n]
}

// FirstWord return the first word
func (ln bufioLn) FirstWord() string {
	offset := 0
	inWord := false
	data := ln.Trim()

	for i, c := range data {
		switch c {
		case ' ', '\t':
			if inWord {
				return string(data[offset:i])
			}
			inWord = false
		default:
			if !inWord {
				offset = i
			}
			inWord = true
		}
	}
	return string(data[offset:])
}

// ParseInt parses an int
func (ln bufioLn) ParseInt() (int64, error) {
	data := ln.Trim()
	if len(data) < 2 {
		return 0, protoErrorf("Protocol error: expected ':', got ' '")
	} else if data[0] != ':' {
		return 0, protoErrorf("Protocol error: expected ':', got '%s'", string(data[0]))
	}

	n, m := int64(0), int64(1)
	for i, c := range data[1:] {
		if c >= '0' && c <= '9' {
			n = n*10 + int64(c-'0')
		} else if c == '-' && i == 0 {
			m = -1
		} else {
			return 0, errNotAnInteger
		}
	}
	return n * m, nil
}

// ParseMessage converts the line to a string
func (ln bufioLn) ParseMessage(prefix byte) (string, error) {
	data := ln.Trim()
	if len(data) < 1 {
		return "", protoErrorf("Protocol error: expected '%s', got ' '", string(prefix))
	} else if data[0] != prefix {
		return "", protoErrorf("Protocol error: expected '%s', got '%s'", string(prefix), string(data[0]))
	}

	return string(data[1:]), nil
}

// ParseSize parses a size with prefix
func (ln bufioLn) ParseSize(prefix byte, fallback error) (int, error) {
	data := ln.Trim()

	if len(data) == 0 {
		return 0, protoErrorf("Protocol error: expected '%s', got ' '", string(prefix))
	} else if data[0] != prefix {
		return 0, protoErrorf("Protocol error: expected '%s', got '%s'", string(prefix), string(data[0]))
	} else if len(data) < 2 {
		return 0, fallback
	}

	n := 0
	for _, c := range data[1:] {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
		} else {
			return 0, fallback
		}
	}
	if n < 0 {
		return 0, fallback
	}
	return n, nil
}

// --------------------------------------------------------------------

type bufioW struct {
	wr    io.Writer
	buf   []byte
	dirty bool
}

// Buffered returns the number of buffered bytes
func (b *bufioW) Buffered() int {
	return len(b.buf)
}

// AppendArrayLen appends an array header to the output buffer
func (b *bufioW) AppendArrayLen(n int) {
	b.appendSize('*', n)
}

// AppendBytes appends bulk bytes to the output buffer
func (b *bufioW) AppendBytes(p []byte) {
	b.appendSize('$', len(p))
	b.buf = append(b.buf, p...)
	b.buf = append(b.buf, binCRLF...)
}

// AppendString appends a bulk string to the output buffer
func (b *bufioW) AppendString(s string) {
	b.appendSize('$', len(s))
	b.buf = append(b.buf, s...)
	b.buf = append(b.buf, binCRLF...)
}

// AppendInlineBytes appends inline bytes to the output buffer
func (b *bufioW) AppendInlineBytes(p []byte) {
	b.buf = append(b.buf, '+')
	b.buf = append(b.buf, p...)
	b.buf = append(b.buf, binCRLF...)
}

// AppendInlineString appends an inline string to the output buffer
func (b *bufioW) AppendInlineString(s string) {
	b.buf = append(b.buf, '+')
	b.buf = append(b.buf, s...)
	b.buf = append(b.buf, binCRLF...)
}

// AppendError appends an error message to the output buffer
func (b *bufioW) AppendError(msg string) {
	b.buf = append(b.buf, '-')
	b.buf = append(b.buf, msg...)
	b.buf = append(b.buf, binCRLF...)
}

// AppendInt appends a numeric response to the output buffer
func (b *bufioW) AppendInt(n int64) {
	switch n {
	case 0:
		b.buf = append(b.buf, binZERO...)
	case 1:
		b.buf = append(b.buf, binONE...)
	default:
		b.buf = append(b.buf, ':')
		b.buf = append(b.buf, strconv.FormatInt(n, 10)...)
		b.buf = append(b.buf, binCRLF...)
	}
}

// AppendNil appends a nil-value to the output buffer
func (b *bufioW) AppendNil() {
	b.buf = append(b.buf, binNIL...)
}

// AppendOK appends "OK" to the output buffer
func (b *bufioW) AppendOK() {
	b.buf = append(b.buf, binOK...)
}

// WriteFromN flushes the existing buffer and read n bytes from the reader directly to
// the client connection.
func (b *bufioW) WriteFromN(r io.Reader, n int) error {
	b.appendSize('$', n)
	if start := len(b.buf); cap(b.buf)-start >= n+2 {
		b.buf = b.buf[:start+n]
		if _, err := io.ReadFull(r, b.buf[start:]); err != nil {
			return err
		}

		b.buf = append(b.buf, binCRLF...)
		return nil
	}

	if err := b.Flush(); err != nil {
		return err
	}
	b.buf = b.buf[:cap(b.buf)]
	_, err := io.CopyBuffer(b.wr, io.LimitReader(r, int64(n)), b.buf)
	b.buf = b.buf[:0]
	if err != nil {
		return err
	}

	b.buf = append(b.buf, binCRLF...)
	return nil
}

// Flush flushes pending buffer
func (b *bufioW) Flush() error {
	if len(b.buf) == 0 {
		return nil
	}

	if _, err := b.wr.Write(b.buf); err != nil {
		return err
	}

	b.buf = b.buf[:0]
	b.dirty = true
	return nil
}

// Reset resets the writer with an new interface
func (b *bufioW) Reset(w io.Writer) {
	b.reset(b.buf, w)
}

func (b *bufioW) appendSize(c byte, n int) {
	b.buf = append(b.buf, c)
	b.buf = append(b.buf, strconv.Itoa(n)...)
	b.buf = append(b.buf, binCRLF...)
}

func (b *bufioW) reset(buf []byte, wr io.Writer) {
	*b = bufioW{buf: buf[:0], wr: wr}
}
