package redeo

import (
	"bufio"
	"context"
	"net"
	"sync"
	"sync/atomic"
)

var clientInc = uint64(0)
var clientPool sync.Pool

// Client contains information about a client connection
type Client struct {
	id uint64
	cn net.Conn

	rd *bufio.Reader
	wr *ResponseBuffer

	buf []byte
	ctx context.Context

	closed bool
}

func newClient(cn net.Conn) *Client {
	var c *Client

	if v := clientPool.Get(); v != nil {
		c = v.(*Client)
		c.rd.Reset(cn)
		c.wr.reset(cn)
		c.buf = c.buf[:0]
	} else {
		c = new(Client)
		c.rd = bufio.NewReader(cn)
		c.wr = NewResponseBuffer(cn)
	}
	c.id = atomic.AddUint64(&clientInc, 1)
	c.cn = cn
	return c
}

// ID return the unique client id
func (c *Client) ID() uint64 { return c.id }

// Context return the client context
func (c *Client) Context() context.Context {
	if c.ctx != nil {
		return c.ctx
	}
	return context.Background()
}

// SetContext sets the client's context
func (c *Client) SetContext(ctx context.Context) {
	c.ctx = ctx
}

// RemoteAddr return the remote client address
func (c *Client) RemoteAddr() net.Addr {
	return c.cn.RemoteAddr()
}

// Close will disconnect as soon as all pending replies have been written
// to the client
func (c *Client) Close() {
	c.closed = true
}

func (c *Client) eachCommand(fn func(*Command) error) error {
	for hasMore := true; hasMore; hasMore = (c.rd.Buffered() != 0) {
		cmd, err := readCommand(c.rd, c)
		if err != nil {
			return err
		}
		if err := fn(cmd); err != nil {
			return err
		}
		cmd.release()
	}
	return nil
}

func (c *Client) release() {
	_ = c.cn.Close()
	clientPool.Put(c)
}
