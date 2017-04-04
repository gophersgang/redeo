package client

import (
	"io"
	"net"

	"github.com/bsm/redeo/resp"
)

// Conn wraps a single network connection and exposes
// common read/write methods.
type Conn interface {
	// PeekType returns the type of the next response block
	PeekType() (resp.ResponseType, error)
	// ReadNil reads a nil value
	ReadNil() error
	// ReadBulk reads a bulk value
	ReadBulk() ([]byte, error)
	// ReadBulkString reads a bulk value as string
	ReadBulkString() (string, error)
	// ReadInt reads an int value
	ReadInt() (int64, error)
	// ReadArrayLen reads the array length
	ReadArrayLen() (int, error)
	// ReadError reads an error string
	ReadError() (string, error)
	// ReadStatus reads a status string
	ReadStatus() (string, error)
	// WriteCmd writes a full command as part of a pipeline. To execute the pipeline,
	// you must call Flush.
	WriteCmd(cmd string, args ...[]byte)
	// WriteCmdString writes a full command as part of a pipeline. To execute the pipeline,
	// you must call Flush.
	WriteCmdString(cmd string, args ...string)
	// WriteMultiBulkSize is a low-level method to write a multibulk size.
	// For normal operation, use WriteCmd or WriteCmdString.
	WriteMultiBulkSize(n int) error
	// WriteBulk is a low-level method to write a bulk.
	// For normal operation, use WriteCmd or WriteCmdString.
	WriteBulk(b []byte)
	// WriteBulkString is a low-level method to write a bulk.
	// For normal operation, use WriteCmd or WriteCmdString.
	WriteBulkString(s string)
	// CopyBulk is a low-level method to copy a large bulk of data directly to the writer.
	// For normal operation, use WriteCmd or WriteCmdString.
	CopyBulk(src io.Reader, n int64) error
	// Flush flushes the output buffer. Call this after you have completed your pipeline
	Flush() error

	madeByRedeo()
}

type conn struct {
	net.Conn

	*resp.RequestWriter
	resp.ResponseReader
}

func (c *conn) madeByRedeo() {}
