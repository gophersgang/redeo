package redeo

import (
	"net"
	"strings"
	"time"

	"github.com/bsm/redeo/resp"
)

// Server configuration
type Server struct {
	config   *Config
	info     *ServerInfo
	commands map[string]Handler
}

// NewServer creates a new server instance
func NewServer(config *Config) *Server {
	if config == nil {
		config = new(Config)
	}

	return &Server{
		config:   config,
		info:     newServerInfo(),
		commands: make(map[string]Handler),
	}
}

// Info returns the server info registry
func (srv *Server) Info() *ServerInfo { return srv.info }

// Handle registers a handler for a command.
// Not thread-safe, don't call from multiple goroutines
func (srv *Server) Handle(name string, handler Handler) {
	srv.commands[strings.ToLower(name)] = handler
}

// HandleFunc registers a handler callback for a command
func (srv *Server) HandleFunc(name string, callback HandlerFunc) {
	srv.Handle(name, Handler(callback))
}

// Serve accepts incoming connections on a listener, creating a
// new service goroutine for each.
func (srv *Server) Serve(lis net.Listener) error {
	for {
		cn, err := lis.Accept()
		if err != nil {
			return err
		}

		if ka := srv.config.TCPKeepAlive; ka > 0 {
			if tc, ok := cn.(*net.TCPConn); ok {
				tc.SetKeepAlive(true)
				tc.SetKeepAlivePeriod(ka)
			}
		}

		go srv.serveClient(newClient(cn))
	}
}

// Starts a new session, serving client
func (srv *Server) serveClient(c *Client) {
	// Release client on exit
	defer c.release()

	// Register client
	srv.info.register(c)
	defer srv.info.deregister(c.id)

	// Init request/response loop
	for !c.closed {
		// set deadline
		if d := srv.config.Timeout; d > 0 {
			c.cn.SetDeadline(time.Now().Add(d))
		}

		// perform pipeline
		if err := c.eachCommand(func(cmd *resp.Command) error {
			srv.perform(c, cmd)

			if n := c.wr.Buffered(); n >= resp.MaxBufferSize {
				return c.wr.Flush()
			}
			return nil
		}); err != nil {
			c.wr.AppendError("ERR " + err.Error())

			if !resp.IsProtocolError(err) {
				_ = c.wr.Flush()
				return
			}
		}

		// flush buffer, return on errors
		if err := c.wr.Flush(); err != nil {
			return
		}
	}
}

func (srv *Server) perform(c *Client, cmd *resp.Command) {
	// find handler
	handler, ok := srv.commands[strings.ToLower(cmd.Name)]
	if !ok {
		c.wr.AppendError(UnknownCommand(cmd.Name))
		return
	}

	// register call
	srv.info.command(c.id, cmd.Name)

	// serve command
	handler.ServeRedeo(c.wr, cmd)
}
