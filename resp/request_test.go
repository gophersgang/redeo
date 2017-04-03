package resp_test

import (
	"bytes"
	"io"
	"strings"

	"github.com/bsm/redeo/resp"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("RequestReader", func() {

	setup := func(s string) *resp.RequestReader {
		return resp.NewRequestReader(bytes.NewBufferString(s))
	}

	It("should read inline requests", func() {
		r := setup("PING\r\nEcHO   HeLLO   \r\n")

		cmd, err := r.ReadCmd(nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(cmd).To(MatchCommand("PING"))
		Expect(cmd.ArgN()).To(Equal(0))
		Expect(cmd.Arg(0)).To(BeNil())

		cmd, err = r.ReadCmd(cmd)
		Expect(err).NotTo(HaveOccurred())
		Expect(cmd).To(MatchCommand("EcHO", "HeLLO"))
		Expect(cmd.ArgN()).To(Equal(1))
		Expect(cmd.Arg(0)).To(Equal(resp.CommandArgument("HeLLO")))
		Expect(cmd.Arg(1)).To(BeNil())

		cmd, err = r.ReadCmd(cmd)
		Expect(err).To(MatchError("EOF"))
		Expect(cmd).To(MatchCommand(""))
	})

	It("should read multi-bulk requests", func() {
		r := setup("*1\r\n$4\r\nPING\r\n*2\r\n$4\r\nEcHO\r\n$5\r\nHeLLO\r\n")

		cmd, err := r.ReadCmd(nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(cmd).To(MatchCommand("PING"))
		Expect(cmd.ArgN()).To(Equal(0))
		Expect(cmd.Arg(0)).To(BeNil())

		cmd, err = r.ReadCmd(cmd)
		Expect(err).NotTo(HaveOccurred())
		Expect(cmd).To(MatchCommand("EcHO", "HeLLO"))
		Expect(cmd.ArgN()).To(Equal(1))
		Expect(cmd.Arg(0)).To(Equal(resp.CommandArgument("HeLLO")))
		Expect(cmd.Arg(1)).To(BeNil())

		cmd, err = r.ReadCmd(cmd)
		Expect(err).To(MatchError("EOF"))
		Expect(cmd).To(MatchCommand(""))
	})

	It("should deal with inconsistent lengths just like Redis", func() {
		r := setup("*1\r\n$4\r\nPING123\r\n*1\r\n$4\r\nPING\r\n")

		cmd, err := r.ReadCmd(nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(cmd).To(MatchCommand("PING"))

		cmd, err = r.ReadCmd(cmd)
		Expect(err).NotTo(HaveOccurred())
		Expect(cmd).To(MatchCommand("3"))

		cmd, err = r.ReadCmd(cmd)
		Expect(err).NotTo(HaveOccurred())
		Expect(cmd).To(MatchCommand("PING"))
	})

	It("should read commands that are larger than the buffer", func() {
		r := setup("*2\r\n$4\r\nECHO\r\n$262144\r\n" + strings.Repeat("x", 262144) + "\r\n")

		cmd, err := r.ReadCmd(nil)
		Expect(err).NotTo(HaveOccurred())
		Expect(cmd.Name).To(Equal("ECHO"))
		Expect(cmd.ArgN()).To(Equal(1))
		Expect(len(cmd.Arg(0))).To(Equal(262144))
	})

	DescribeTable("should read commands",
		func(s string, exp [][]string) {
			var act [][]string

			r := setup(s)
			for {
				cmd, err := r.ReadCmd(nil)
				if err == io.EOF {
					break
				}
				Expect(err).NotTo(HaveOccurred())
				act = append(act, cmdToSlice(cmd))
			}
			Expect(act).To(Equal(exp))
		},

		Entry("inline requests",
			"PING\r\n",
			[][]string{
				{"PING"},
			}),
		Entry("multiple inline requests with args",
			"  ECHO HELLO  \r\nECHO   WORLD   \r\n",
			[][]string{
				{"ECHO", "HELLO"},
				{"ECHO", "WORLD"},
			}),
		Entry("blank multi-bulks",
			"*0\r\nPING\r\n",
			[][]string{
				{"PING"},
			}),
		Entry("blank commands",
			"*1\r\n$0\r\n\r\n",
			[][]string{
				{""},
			}),
		Entry("blank commands without line-break",
			"*1\r\n$0\r\n",
			[][]string{
				{""},
			}),
		Entry("blank commands without line-break followed by inline command",
			"*1\r\n$0\r\nPING\r\n",
			[][]string{
				{""},
				{"NG"},
			}),
		Entry("extra line breaks",
			"\r\nPING\r\n\r\nPING\r\n",
			[][]string{
				{"PING"},
				{"PING"},
			}),
	)

	DescribeTable("should reject proto errors",
		func(s string, msg string) {
			r := setup(s)
			_, err := r.ReadCmd(nil)
			Expect(err).To(MatchError(msg))
		},

		Entry("blank multi-bulk len", "*\r\n", "Protocol error: invalid multibulk length"),
		Entry("bad multi-bulk len", "*x\r\n", "Protocol error: invalid multibulk length"),
		Entry("inline inside multi-bulk", "*1\r\nPING\r\n", "Protocol error: expected '$', got 'P'"),
		Entry("bad bulk length", "*1\r\n$x\r\n", "Protocol error: invalid bulk length"),
		Entry("negative bulk length", "*1\r\n$-1\r\n", "Protocol error: invalid bulk length"),
	)

	DescribeTable("should peek commands",
		func(s string, exp string) {
			r := setup(s)
			name, err := r.PeekCmd()
			Expect(err).NotTo(HaveOccurred())
			Expect(name).To(Equal(exp))
		},

		Entry("inline requests",
			"PING\r\n", "PING"),
		Entry("multiple inline requests with args",
			"  ECHO HELLO  \r\n", "ECHO"),
		Entry("blank multi-bulks",
			"*0\r\nPING\r\n", "PING"),
		Entry("multi-bulks",
			"*1\r\n$4\r\nPING\r\n", "PING"),
	)

})

var _ = Describe("RequestWriter", func() {
	var buf = new(bytes.Buffer)

	setup := func() *resp.RequestWriter {
		buf.Reset()
		return resp.NewRequestWriter(buf)
	}

	DescribeTable("should write string commands",
		func(cmd string, args []string, exp string) {
			w := setup()
			w.WriteCmdString(cmd, args...)
			Expect(w.Buffered()).To(Equal(len(exp)))
			Expect(buf.Len()).To(Equal(0))

			Expect(w.Flush()).To(Succeed())
			Expect(w.Buffered()).To(Equal(0))
			Expect(buf.String()).To(Equal(exp))
		},

		Entry("simple commands", "PING", nil,
			"*1\r\n$4\r\nPING\r\n"),
		Entry("commands with arguments", "eCHo", []string{"heLLo"},
			"*2\r\n$4\r\neCHo\r\n$5\r\nheLLo\r\n"),
	)

	DescribeTable("should write byte commands",
		func(cmd string, args [][]byte, exp string) {
			w := setup()
			w.WriteCmd(cmd, args...)
			Expect(w.Buffered()).To(Equal(len(exp)))
			Expect(buf.Len()).To(Equal(0))

			Expect(w.Flush()).To(Succeed())
			Expect(w.Buffered()).To(Equal(0))
			Expect(buf.String()).To(Equal(exp))
		},

		Entry("simple commands", "PING", nil,
			"*1\r\n$4\r\nPING\r\n"),
		Entry("commands with arguments", "eCHo", [][]byte{[]byte("heLLo")},
			"*2\r\n$4\r\neCHo\r\n$5\r\nheLLo\r\n"),
	)

	It("should allow to buffer arguments from readers", func() {
		rd := bytes.NewBufferString("this is a stream of data")
		w := setup()
		Expect(w.WriteMultiBulkSize(3)).To(Succeed())
		w.WriteBulkString("PUT")
		w.WriteBulkString("key")
		Expect(w.Buffered()).To(Equal(22))
		Expect(buf.Len()).To(Equal(0))

		Expect(w.WriteFromN(rd, 12)).To(Succeed())
		Expect(w.Buffered()).To(Equal(41))
		Expect(buf.Len()).To(Equal(0))

		Expect(w.Flush()).To(Succeed())
		Expect(w.Buffered()).To(Equal(0))
		Expect(buf.Len()).To(Equal(41))
		Expect(buf.String()).To(Equal("*3\r\n$3\r\nPUT\r\n$3\r\nkey\r\n$12\r\nthis is a st\r\n"))
	})

	It("should copy oversize arguments directly from reader", func() {
		rd := bytes.NewBufferString(strings.Repeat("x", 100000))
		w := setup()
		Expect(w.WriteMultiBulkSize(3)).To(Succeed())
		w.WriteBulkString("PUT")
		w.WriteBulkString("key")
		Expect(w.Buffered()).To(Equal(22))
		Expect(buf.Len()).To(Equal(0))

		Expect(w.WriteFromN(rd, 80000)).To(Succeed())
		Expect(w.Buffered()).To(Equal(2))
		Expect(buf.Len()).To(Equal(80030))
		Expect(w.Flush()).To(Succeed())
		Expect(buf.Len()).To(Equal(80032))
	})

})
