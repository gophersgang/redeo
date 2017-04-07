// Package client implements a minimalist client
// for working with redis servers.
package client

import (
	"net"
	"sync"

	"github.com/bsm/pool"
	"github.com/bsm/redeo/resp"
)

// Client is a pooled minimalist redis client
type Client struct {
	conns   *pool.Pool
	readers sync.Pool
	writers sync.Pool
}

// New initializes a new client with a custom dialer
func New(opt *pool.Options, dialer func() (net.Conn, error)) (*Client, error) {
	if dialer == nil {
		dialer = func() (net.Conn, error) {
			return net.Dial("tcp", "127.0.0.1:6379")
		}
	}

	conns, err := pool.New(opt, dialer)
	if err != nil {
		return nil, err
	}

	return &Client{
		conns: conns,
	}, nil
}

// Get returns a connection
func (c *Client) Get() (Conn, error) {
	cn, err := c.conns.Get()
	if err != nil {
		return nil, err
	}

	return &conn{
		Conn: cn,

		RequestWriter:  c.newRequestWriter(cn),
		ResponseReader: c.newResponseReader(cn),
	}, nil
}

// Put allows to return a connection back to the pool.
// Call this method after every call/pipeline.
// Do not use the connection again after this method
// is triggered.
func (c *Client) Put(cn Conn) {
	cs, ok := cn.(*conn)
	if !ok {
		return
	} else if cs.failed {
		_ = cs.Close()
		return
	}

	c.writers.Put(cs.RequestWriter)
	c.readers.Put(cs.ResponseReader)
	c.conns.Put(cs.Conn)
}

// Close closes the client and all underlying connections
func (c *Client) Close() error {
	return c.conns.Close()
}

func (c *Client) newRequestWriter(cn net.Conn) *resp.RequestWriter {
	if v := c.writers.Get(); v != nil {
		w := v.(*resp.RequestWriter)
		w.Reset(cn)
		return w
	}
	return resp.NewRequestWriter(cn)
}

func (c *Client) newResponseReader(cn net.Conn) resp.ResponseReader {
	if v := c.readers.Get(); v != nil {
		r := v.(resp.ResponseReader)
		r.Reset(cn)
		return r
	}
	return resp.NewResponseReader(cn)
}
