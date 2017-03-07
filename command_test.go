package redeo

import (
	"bufio"
	"io"
	"strings"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("Command", func() {

	DescribeTable("readCommand (successfully)",
		func(msg string, exp *Command) {
			bio := bufio.NewReader(strings.NewReader(msg))
			cmd, err := readCommand(bio, &Client{})
			Expect(err).NotTo(HaveOccurred())

			exp.client = cmd.client
			Expect(cmd).To(Equal(exp))

			_, err = bio.Peek(1)
			Expect(err).To(Equal(io.EOF))
		},
		Entry("inline ping", "PiNg\r\n", &Command{Name: "ping"}),
		Entry("bulk ping", "*1\r\n$4\r\nPiNg\r\n", &Command{Name: "ping"}),
		Entry("get", "*2\r\n$3\r\nGET\r\n$2\r\nXy\r\n", &Command{Name: "get", Args: []string{"Xy"}}),
		Entry("set", "*3\r\n$3\r\nSET\r\n$5\r\nk\r\ney\r\n$5\r\nva\r\nl\r\n", &Command{Name: "set", Args: []string{"k\r\ney", "va\r\nl"}}),
	)

	DescribeTable("readCommand (failures)",
		func(msg string, exp string) {
			bio := bufio.NewReader(strings.NewReader(msg))
			cmd, err := readCommand(bio, &Client{})
			Expect(cmd).To(BeNil())
			Expect(err).To(MatchError(exp))
		},
		Entry("blank", "", "EOF"),
		Entry("blank with CRLF", "\r\n", "EOF"),
		Entry("no multibulk length", "*x\r\n", "Protocol error: invalid multibulk length"),
		Entry("no bulk length", "*1\r\nping\r\n", "Protocol error: expected '$', got 'p'"),
		Entry("invalid bulk length", "*1\r\n$x\r\nping\r\n", "Protocol error: invalid bulk length"),
		Entry("missing multi-bulk", "*2\r\n$3\r\nGET\r\n", "EOF"),
		Entry("truncated argument", "*2\r\n$3\r\nGE", "EOF"),
	)

	It("should parse chunks", func() {
		msg := "*3\r\n$3\r\nset\r\n$1\r\nx\r\n$1024\r\n" + strings.Repeat("x", 1024) + "\r\n"
		bio := bufio.NewReader(strings.NewReader(msg))

		cmd, err := readCommand(bio, &Client{})
		Expect(err).NotTo(HaveOccurred())
		Expect(cmd).NotTo(BeNil())
		Expect(cmd.Args).To(HaveLen(2))
		Expect(cmd.Args[1]).To(HaveLen(1024))
		Expect(cmd.Args[1][1020:]).To(Equal("xxxx"))
	})

	It("should support pipelining", func() {
		msg := "PiNg\r\n*2\r\n$3\r\nGET\r\n$2\r\nXy\r\n"
		bio := bufio.NewReader(strings.NewReader(msg))

		cmd, err := readCommand(bio, &Client{})
		Expect(err).NotTo(HaveOccurred())
		Expect(cmd.Name).To(Equal("ping"))

		cmd, err = readCommand(bio, &Client{})
		Expect(err).NotTo(HaveOccurred())
		Expect(cmd.Name).To(Equal("get"))

		_, err = readCommand(bio, &Client{})
		Expect(err).To(MatchError("EOF"))
	})

})

func Benchmark_readCommand_Inline(b *testing.B) {
	c := &Client{}
	r := strings.NewReader("")
	bio := bufio.NewReader(r)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.Reset("ping\r\n")
		bio.Reset(r)

		cmd, err := readCommand(bio, c)
		if err != nil {
			b.Fatal(err)
		}
		cmd.release()
	}
}

func Benchmark_readCommand_Bulk(b *testing.B) {
	c := &Client{}
	r := strings.NewReader("")
	bio := bufio.NewReader(r)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.Reset("*2\r\n$3\r\nget\r\n$1\r\nx\r\n")
		bio.Reset(r)

		cmd, err := readCommand(bio, c)
		if err != nil {
			b.Fatal(err)
		}
		cmd.release()
	}
}
