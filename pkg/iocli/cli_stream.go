package iocli

import (
	"fmt"
	"io"
	"strings"

	"github.com/docker/cli/cli/streams"
)

type CliStream struct {
	inputStream  *streams.In
	outputStream *streams.Out
	auxStream    *streams.Out
	errorStream  io.Writer
}

var CLI = &CliStream{}

func NewCliStream(cin io.ReadCloser, cout io.Writer, cerr io.Writer) *CliStream {
	return &CliStream{
		inputStream:  streams.NewIn(cin),
		outputStream: streams.NewOut(cout),
		auxStream:    streams.NewOut(cerr),
		errorStream:  cerr,
	}
}

func (c *CliStream) InputStream() *streams.In {
	return c.inputStream
}

func (c *CliStream) OutputStream() *streams.Out {
	return c.outputStream
}

func (c *CliStream) AuxStream() *streams.Out {
	return c.auxStream
}

func (c *CliStream) ErrorStream() io.Writer {
	return c.errorStream
}

func (c *CliStream) SetQuiet(v bool) {
	if v {
		c.auxStream = streams.NewOut(io.Discard)
	} else {
		c.auxStream = streams.NewOut(c.errorStream)
	}
}

func (c *CliStream) PrintOut(format string, a ...any) {
	fmt.Fprintf(c.OutputStream(), format, a...)
}

func (c *CliStream) PrintErr(format string, a ...any) {
	fmt.Fprintf(c.ErrorStream(), format, a...)
}

func (c *CliStream) PrintAux(format string, a ...any) {
	fmt.Fprintf(c.AuxStream(), format, a...)
}

type StatusError struct {
	status string
	code   int
}

var _ error = StatusError{}

func NewStatusError(code int, format string, a ...any) StatusError {
	status := strings.TrimSuffix(fmt.Sprintf(format, a...), ".") + "."
	return StatusError{
		code:   code,
		status: strings.ToUpper(status[:1]) + status[1:],
	}
}

func WrapStatusError(err error) error {
	if err == nil {
		return nil
	}
	return NewStatusError(1, err.Error())
}

func (e StatusError) Error() string {
	return e.status
}

func (e StatusError) Code() int {
	return e.code
}
