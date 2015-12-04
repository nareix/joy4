
package ts

import (
	_ "fmt"
	"io"
)

type TSWriter struct {
	W io.Writer
}

type PSIWriter struct {
	W *TSWriter
}

func (self PSIWriter) Write(b []byte) (err error) {
	return
}

func (self PSIWriter) Finish() (err error) {
	return
}

type PESWriter struct {
	W io.Writer
}

type SimpleH264Writer struct {
	W io.Writer
	headerHasWritten bool
}

func WritePAT(w io.Writer, self PAT) (err error) {
	return
}

func (self *SimpleH264Writer) WriteSample(data []byte) (err error) {
	return
}

func (self *SimpleH264Writer) WriteNALU(data []byte) (err error) {
	return
}

