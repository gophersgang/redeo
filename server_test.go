package redeo

import (
	"net"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Server", func() {
	var subject *Server

	var (
		pong  = func(w *ResponseBuffer, _ *Command) { w.AppendInlineString("PONG") }
		blank = func(w *ResponseBuffer, _ *Command) {}
		echo  = func(w *ResponseBuffer, req *Command) {
			if len(req.Args) != 1 {
				w.AppendError(WrongNumberOfArgs(req.Name))
				return
			}
			w.AppendString(req.Args[0])
		}
		flush = func(w *ResponseBuffer, _ *Command) {
			w.AppendOK()
			w.Flush()
		}
	)

	var runServer = func(srv *Server, fn func(net.Conn, []byte)) {
		lis, err := net.Listen("tcp", "127.0.0.1:0")
		Expect(err).NotTo(HaveOccurred())
		defer lis.Close()

		// start listening
		go srv.Serve(lis)

		// connect client
		conn, err := net.Dial("tcp", lis.Addr().String())
		Expect(err).NotTo(HaveOccurred())
		defer conn.Close()

		fn(conn, make([]byte, 20000))
	}

	BeforeEach(func() {
		subject = NewServer(&Config{
			Timeout: 100 * time.Millisecond,
		})
		subject.HandleFunc("pInG", pong)
		subject.HandleFunc("blank", blank)
		subject.HandleFunc("echo", echo)
		subject.HandleFunc("flush", flush)
	})

	It("should register handlers", func() {
		Expect(subject.commands).To(HaveLen(4))
		Expect(subject.commands).To(HaveKey("ping"))
	})

	It("should serve", func() {
		runServer(subject, func(conn net.Conn, buf []byte) {
			_, err := conn.Write([]byte("PING\r\n"))
			Expect(err).NotTo(HaveOccurred())

			n, err := conn.Read(buf)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(buf[:n])).To(Equal("+PONG\r\n"))

			info := subject.Info()
			Expect(info.NumClients()).To(Equal(1))
			Expect(info.TotalCommands()).To(Equal(int64(1)))
			Expect(info.TotalConnections()).To(Equal(int64(1)))
			Expect(info.ClientInfo()[0].LastCmd).To(Equal("ping"))

			_, err = conn.Write([]byte("*2\r\n$4\r\necho\r\n$10000\r\n" + strings.Repeat("x", 10000) + "\r\n"))
			Expect(err).NotTo(HaveOccurred())

			n, err = conn.Read(buf)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(buf[:8])).To(Equal("$10000\r\n"))

			info = subject.Info()
			Expect(info.NumClients()).To(Equal(1))
			Expect(info.TotalCommands()).To(Equal(int64(2)))
			Expect(info.TotalConnections()).To(Equal(int64(1)))
			Expect(info.ClientInfo()[0].LastCmd).To(Equal("echo"))
		})
	})

	It("should handle pipelines", func() {
		runServer(subject, func(conn net.Conn, buf []byte) {
			_, err := conn.Write([]byte("*1\r\n$4\r\nPING\r\n*1\r\n$4\r\nPING\r\n*1\r\n$4\r\nPING\r\n"))
			Expect(err).NotTo(HaveOccurred())

			n, err := conn.Read(buf)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(buf[:n])).To(Equal("+PONG\r\n+PONG\r\n+PONG\r\n"))
		})
	})

	It("should have a default response", func() {
		runServer(subject, func(conn net.Conn, buf []byte) {
			_, err := conn.Write([]byte("BLANK\r\n"))
			Expect(err).NotTo(HaveOccurred())

			n, err := conn.Read(buf)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(buf[:n])).To(Equal("+OK\r\n"))

			_, err = conn.Write([]byte("FLUSH\r\n"))
			Expect(err).NotTo(HaveOccurred())

			n, err = conn.Read(buf)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(buf[:n])).To(Equal("+OK\r\n"))

			_, err = conn.Read(buf)
			Expect(err).To(MatchError("EOF"))
		})
	})

	It("should handle invalid commands", func() {
		runServer(subject, func(conn net.Conn, buf []byte) {
			_, err := conn.Write([]byte("NOOP\r\n"))
			Expect(err).NotTo(HaveOccurred())

			n, err := conn.Read(buf)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(buf[:n])).To(Equal("-ERR unknown command 'noop'\r\n"))

			// connection should still be open
			_, err = conn.Write([]byte("PING\r\n"))
			Expect(err).NotTo(HaveOccurred())
		})
	})

	It("should handle invalid commands in pipelines", func() {
		runServer(subject, func(conn net.Conn, buf []byte) {
			_, err := conn.Write([]byte("*1\r\n$4\r\nPING\r\n*1\r\n$3\r\nBAD\r\n*1\r\n$4\r\nPING\r\n"))
			Expect(err).NotTo(HaveOccurred())

			n, err := conn.Read(buf)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(buf[:n])).To(Equal("+PONG\r\n-ERR unknown command 'bad'\r\n+PONG\r\n"))
		})
	})

	It("should handle client errors", func() {
		runServer(subject, func(conn net.Conn, buf []byte) {
			_, err := conn.Write([]byte("ECHO\r\n"))
			Expect(err).NotTo(HaveOccurred())

			n, err := conn.Read(buf)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(buf[:n])).To(Equal("-ERR wrong number of arguments for 'echo' command\r\n"))

			// connection should still be open
			_, err = conn.Write([]byte("PING\r\n"))
			Expect(err).NotTo(HaveOccurred())
		})
	})

	It("should handle client errors in pipelines", func() {
		runServer(subject, func(conn net.Conn, buf []byte) {
			_, err := conn.Write([]byte("PING\r\nECHO\r\nPING\r\n"))
			Expect(err).NotTo(HaveOccurred())

			n, err := conn.Read(buf)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(buf[:n])).To(Equal("+PONG\r\n-ERR wrong number of arguments for 'echo' command\r\n+PONG\r\n"))
		})
	})

	It("should handle protocol errors", func() {
		runServer(subject, func(conn net.Conn, buf []byte) {
			_, err := conn.Write([]byte("*x\r\n"))
			Expect(err).NotTo(HaveOccurred())

			n, err := conn.Read(buf)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(buf[:n])).To(Equal("-ERR Protocol error: invalid multibulk length\r\n"))

			// connection should still be open
			_, err = conn.Write([]byte("PING\r\n"))
			Expect(err).NotTo(HaveOccurred())
		})
	})

	It("should handle protocol errors in pipelines", func() {
		runServer(subject, func(conn net.Conn, buf []byte) {
			_, err := conn.Write([]byte("*1\r\n$4\r\nPING\r\n*1\r\n$x\r\nPING\r\n*1\r\n$4\r\nPING\r\n"))
			Expect(err).NotTo(HaveOccurred())

			n, err := conn.Read(buf)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(buf[:n])).To(Equal("+PONG\r\n-ERR Protocol error: invalid bulk length\r\n"))
		})
	})

	It("should close connections on EOF errors", func() {
		runServer(subject, func(conn net.Conn, buf []byte) {
			_, err := conn.Write([]byte("*1\r\n$4\r\nPI"))
			Expect(err).NotTo(HaveOccurred())

			// connection should be closed
			_, err = conn.Read(buf)
			Expect(err).To(MatchError("EOF"))
		})
	})

})

func BenchmarkServer(b *testing.B) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		b.Fatal(err)
	}
	defer lis.Close()

	srv := NewServer(nil)
	srv.HandleFunc("ping", func(w *ResponseBuffer, _ *Command) {
		w.AppendInlineString("PONG")
	})

	go srv.Serve(lis)

	conn, err := net.Dial("tcp", lis.Addr().String())
	if err != nil {
		b.Fatal(err)
	}
	defer conn.Close()

	buf := make([]byte, 1024)
	pipe := []byte("PING\r\nPING\r\nPING\r\n")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := conn.Write(pipe); err != nil {
			b.Fatal(err)
		}
		if n, err := conn.Read(buf); err != nil {
			b.Fatal(err)
		} else if n != 21 {
			b.Fatalf("expected response to be 21 bytes long, not %d", n)
		}
	}
}
