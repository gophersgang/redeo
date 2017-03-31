package resp

import (
	"bytes"
	"fmt"
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
			t = TypeString
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
	if err := b.consume(binNIL[:3]); err != nil {
		return err
	}
	b.DiscardCRLF()
	return nil
}

func (b *bufioR) ReadInt() (int64, error) {
	c, err := b.ReadByte()
	if err != nil {
		return 0, err
	} else if c != ':' {
		return 0, errNotAnInt
	}

	firstByte := true
	n, m := int64(0), int64(1)
	for {
		c, err := b.ReadByte()
		if err != nil {
			return 0, err
		}

		if c >= '0' && c <= '9' {
			n = n*10 + int64(c-'0')
		} else if c == '-' && firstByte {
			m = -1
		} else if (c == '\r' || c == '\n') && !firstByte {
			break
		} else {
			return 0, errNotAnInt
		}
		firstByte = false
	}
	b.DiscardCRLF()
	return n * m, nil
}

func (b *bufioR) ReadByte() (byte, error) {
	c, err := b.PeekByte()
	if err != nil {
		return 0, err
	}

	b.r++
	return c, nil
}

func (b *bufioR) ReadError() (string, error) {
	return b.readLine('-', errNotAnError)
}

func (b *bufioR) ReadStatus() (string, error) {
	return b.readLine('+', errNotAStatus)
}

func (b *bufioR) ReadArrayLen() (int, error) {
	return b.readSize('*', errInvalidMultiBulkLength)
}

func (b *bufioR) ReadStringLen() (int, error) {
	sz, err := b.readSize('$', errInvalidBulkLength)
	if err != nil {
		return 0, err
	} else if sz < 0 {
		return 0, errInvalidBulkLength
	}
	return sz, nil
}

func (b *bufioR) ReadBytes() ([]byte, error) {
	sz, err := b.ReadStringLen()
	if err != nil {
		return nil, err
	}

	if sz < 1 {
		b.skip(2)
		return nil, nil
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

func (b *bufioR) ReadString() (string, error) {
	bb, err := b.ReadBytes()
	return string(bb), err
}

func (b *bufioR) skip(sz int) {
	if b.Buffered() >= sz {
		b.r += sz
	}
}

// Discard reads and discards CRLF
func (b *bufioR) DiscardCRLF() {
	for ; b.r < b.w; b.r++ {
		switch b.buf[b.r] {
		case '\r', '\n':
			// continue
		default:
			return
		}
	}
}

// Reset resets the reader with an new interface
func (b *bufioR) Reset(r io.Reader) {
	b.reset(b.buf, r)
}

func (b *bufioR) consume(data []byte) error {
	for _, x := range data {
		c, err := b.ReadByte()
		if err != nil {
			return err
		} else if c != x {
			return fmt.Errorf("Protocol error: expected '%s', got '%s'", string(x), string(c))
		}
	}
	return nil
}

func (b *bufioR) readSize(prefix byte, invalid error) (int, error) {
	c, err := b.ReadByte()
	if err != nil {
		return 0, fmt.Errorf("Protocol error: expected '%s', got ' '", string(prefix))
	} else if c != prefix {
		return 0, fmt.Errorf("Protocol error: expected '%s', got '%s'", string(prefix), string(c))
	}

	firstByte := true
	n := 0
	for {
		c, err := b.ReadByte()
		if err != nil {
			return 0, err
		}

		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
		} else if (c == '\r' || c == '\n') && !firstByte {
			break
		} else {
			return 0, invalid
		}
		firstByte = false
	}

	b.DiscardCRLF()
	return n, nil
}

func (b *bufioR) readLine(prefix byte, invalid error) (string, error) {
	c, err := b.ReadByte()
	if err != nil {
		return "", err
	} else if c != prefix {
		return "", invalid
	}

	// find the end of the line
	pos := bytes.IndexByte(b.buf[b.r:b.w], '\r')

	// try to read more data into the buffer if not in the buffer
	if pos < 0 {
		if err := b.fill(); err != nil {
			return "", err
		}
		pos = bytes.IndexByte(b.buf[b.r:b.w], '\r')
	}

	// fail if still nothing found
	if pos < 0 {
		return "", invalid
	}

	// read line and advance cursor
	line := string(b.buf[b.r : b.r+pos])
	b.r += pos
	b.DiscardCRLF()
	return line, nil
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

// fill tries to read more data into the buffer
func (b *bufioR) fill() error {
	b.compact()

	n, err := b.rd.Read(b.buf[b.w:])
	b.w += n
	return err
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
