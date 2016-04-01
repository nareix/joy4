
package atom

import (
	"io"
	"fmt"
	"strings"
	"encoding/hex"
)

type Walker interface {
	FilterArrayItem(string,string,int,int) bool
	ArrayLeft(int,int)
	StartStruct(string)
	EndStruct()
	Name(string)
	Int(int)
	Fixed(Fixed)
	String(string)
	Bytes([]byte)
	TimeStamp(TimeStamp)
}

type Dumper struct {
	W io.Writer
	depth int
	name string
	arrlen int
	arridx int
}

func (self Dumper) tab() string {
	return strings.Repeat(" ", self.depth*2)
}

func (self Dumper) println(msg string) {
	fmt.Fprintln(self.W, self.tab()+msg)
}

func (self *Dumper) ArrayLeft(i int, n int) {
	self.println(fmt.Sprintf("... total %d elements", n))
}

func (self *Dumper) FilterArrayItem(name string, field string, i int, n int) bool {
	if n > 20 && i > 20 {
		return false
	}
	return true
}

func (self *Dumper) EndArray() {
}

func (self *Dumper) StartStruct(name string) {
	self.depth++
	self.println(fmt.Sprintf("[%s]", name))
}

func (self *Dumper) EndStruct() {
	self.depth--
}

func (self *Dumper) Name(name string) {
	self.name = name
}

func (self Dumper) Int(val int) {
	self.println(fmt.Sprintf("%s: %d", self.name, val))
}

func (self Dumper) String(val string) {
	self.println(fmt.Sprintf("%s: %s", self.name, val))
}

func (self Dumper) Fixed(val Fixed) {
	self.println(fmt.Sprintf("%s: %d", self.name, FixedToInt(val)))
}

func (self Dumper) Bytes(val []byte) {
	self.println(fmt.Sprintf("%s: %s", self.name, hex.EncodeToString(val)))
}

func (self Dumper) TimeStamp(val TimeStamp) {
	self.println(fmt.Sprintf("%s: %d", self.name, int(val)))
}

