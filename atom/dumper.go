
package atom

import (
	"fmt"
	"strings"
	"encoding/hex"
)

type Walker interface {
	Start()
	Log(string)
	Name(string)
	Int(int)
	Fixed(Fixed)
	String(string)
	Bytes([]byte)
	TimeStamp(TimeStamp)
	End()
}

type Dumper struct {
	depth int
}

func (self *Dumper) Start() {
	self.depth++
}

func (self *Dumper) End() {
	self.depth--
}

func (self Dumper) tab() string {
	return strings.Repeat(" ", self.depth*2)
}

func (self Dumper) Name(name string) {
	fmt.Print(self.tab(), name, ": ")
}

func (self Dumper) Log(msg string) {
	fmt.Println(self.tab()+msg)
}

func (self Dumper) logVal(msg string) {
	fmt.Println(msg)
}

func (self Dumper) Int(val int) {
	self.logVal(fmt.Sprintf("%d", val))
}

func (self Dumper) Fixed(val Fixed) {
	self.logVal(fmt.Sprintf("%d", FixedToInt(val)))
}

func (self Dumper) String(val string) {
	self.logVal(val)
}

func (self Dumper) Bytes(val []byte) {
	self.logVal(hex.EncodeToString(val))
}

func (self Dumper) TimeStamp(val TimeStamp) {
	self.logVal(fmt.Sprintf("%d", int(val)))
}

