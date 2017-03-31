// Package resp implements low-level primitives for dealing
// with RESP (REdis Serialization Protocol). It provides client and
// server side readers and writers.
package resp

import "errors"

type protocolError string

func (p protocolError) Error() string { return string(p) }

// IsProtocolError returns true if the error is a protocol error
func IsProtocolError(err error) bool {
	_, ok := err.(protocolError)
	return ok
}

const (
	errInvalidMultiBulkLength = protocolError("Protocol error: invalid multibulk length")
	errInvalidBulkLength      = protocolError("Protocol error: invalid bulk length")
	errBlankBulkLength        = protocolError("Protocol error: expected '$', got ' '")
)

var (
	errNotAnInt   = errors.New("resp: not an int")
	errNotAnError = errors.New("resp: not an error")
	errNotAStatus = errors.New("resp: not a status")
)

var (
	binCRLF = []byte("\r\n")
	binOK   = []byte("+OK\r\n")
	binZERO = []byte(":0\r\n")
	binONE  = []byte(":1\r\n")
	binNIL  = []byte("$-1\r\n")
)

const defaultBufferSize = 4096

func mkbuf() []byte { return make([]byte, defaultBufferSize) }
