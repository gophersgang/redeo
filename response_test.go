package redeo

import (
	"bytes"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("ResponseBuffer", func() {
	var subject *ResponseBuffer
	var buf = new(bytes.Buffer)

	BeforeEach(func() {
		buf.Reset()
		subject = NewResponseBuffer(buf)
	})

	It("should append bytes", func() {
		subject.AppendBytes([]byte("dAtA"))
		Expect(buf.String()).To(BeEmpty())
		Expect(subject.Flush()).To(Succeed())
		Expect(buf.String()).To(Equal("$4\r\ndAtA\r\n"))
	})

	It("should append strings", func() {
		subject.AppendString("PONG")
		Expect(buf.String()).To(BeEmpty())
		Expect(subject.Flush()).To(Succeed())
		Expect(buf.String()).To(Equal("$4\r\nPONG\r\n"))

		subject.AppendString("日本")
		Expect(subject.Flush()).To(Succeed())
		Expect(buf.String()).To(Equal("$4\r\nPONG\r\n$6\r\n日本\r\n"))
	})

	It("should append inline bytes", func() {
		subject.AppendInlineBytes([]byte("dAtA"))
		Expect(buf.String()).To(BeEmpty())
		Expect(subject.Flush()).To(Succeed())
		Expect(buf.String()).To(Equal("+dAtA\r\n"))
	})

	It("should append inline strings", func() {
		subject.AppendInlineString("PONG")
		Expect(buf.String()).To(BeEmpty())
		Expect(subject.Flush()).To(Succeed())
		Expect(buf.String()).To(Equal("+PONG\r\n"))
	})

	It("should append errors", func() {
		subject.AppendError("WRONGTYPE not a number")
		Expect(buf.String()).To(BeEmpty())
		Expect(subject.Flush()).To(Succeed())
		Expect(buf.String()).To(Equal("-WRONGTYPE not a number\r\n"))
	})

	It("should append ints", func() {
		subject.AppendInt(27)
		Expect(buf.String()).To(BeEmpty())
		Expect(subject.Flush()).To(Succeed())
		Expect(buf.String()).To(Equal(":27\r\n"))

		subject.AppendInt(1)
		Expect(subject.Flush()).To(Succeed())
		Expect(buf.String()).To(Equal(":27\r\n:1\r\n"))
	})

	It("should append nils", func() {
		subject.AppendNil()
		Expect(buf.String()).To(BeEmpty())
		Expect(subject.Flush()).To(Succeed())
		Expect(buf.String()).To(Equal("$-1\r\n"))
	})

	It("should append OK", func() {
		subject.AppendOK()
		Expect(buf.String()).To(BeEmpty())
		Expect(subject.Flush()).To(Succeed())
		Expect(buf.String()).To(Equal("+OK\r\n"))
	})

	It("should copy from readers", func() {
		src := strings.NewReader("this is a streaming data source")
		subject.AppendArrayLen(1)
		Expect(buf.String()).To(BeEmpty())
		Expect(subject.CopyN(src, 16)).To(Succeed())
		Expect(subject.Flush()).To(Succeed())
		Expect(buf.String()).To(Equal("*1\r\n$16\r\nthis is a stream\r\n"))
	})

})
