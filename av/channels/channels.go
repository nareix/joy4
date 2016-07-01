package channels

import (
	"encoding/binary"
	"fmt"
	"github.com/nareix/joy4/av"
	"hash/fnv"
	"io"
	"sync"
	"sync/atomic"
)

func hashParams(params []interface{}) (res uint64, err error) {
	f := fnv.New64()
	for _, p := range params {
		if s, ok := p.(string); ok {
			io.WriteString(f, s)
		} else {
			if err = binary.Write(f, binary.LittleEndian, p); err != nil {
				return
			}
		}
	}
	res = f.Sum64()
	return
}

type Publisher struct {
	h                uint64
	Params           []interface{}
	context          *Context
	streams          []av.CodecData
	closed           bool
	lock             *sync.RWMutex
	cond             *sync.Cond
	subscribersCount int32
	pkt              struct {
		av.Packet
		i int
	}
	ondemand bool
}

func newPublisher() *Publisher {
	pub := &Publisher{}
	pub.lock = &sync.RWMutex{}
	pub.cond = sync.NewCond(pub.lock.RLocker())
	return pub
}

func (self *Publisher) addSubscriber() (sub *Subscriber, err error) {
	self.cond.L.Lock()
	for len(self.streams) == 0 && !self.closed {
		self.cond.Wait()
	}
	self.cond.L.Unlock()

	if self.closed {
		err = fmt.Errorf("publisher closed")
		return
	}

	atomic.AddInt32(&self.subscribersCount, 1)
	sub = &Subscriber{}
	sub.pub = self

	return
}

func (self *Publisher) WriteHeader(streams []av.CodecData) (err error) {
	self.lock.Lock()
	self.streams = streams
	self.lock.Unlock()
	self.cond.Broadcast()
	return
}

func (self *Publisher) WritePacket(i int, pkt av.Packet) (err error) {
	var closed bool

	self.lock.Lock()
	if !self.closed {
		self.pkt.Packet = pkt
		self.pkt.i = i
	} else {
		closed = true
	}
	self.lock.Unlock()

	if closed {
		err = io.EOF
		return
	}

	self.cond.Broadcast()
	return
}

func (self *Publisher) Close() (err error) {
	self.lock.Lock()
	self.closed = true
	self.lock.Unlock()
	self.cond.Broadcast()
	return
}

func (self *Publisher) removeSubscriber() {
	count := atomic.AddInt32(&self.subscribersCount, -1)
	if count == 0 && self.ondemand {
		self.Close()
	}
}

type Subscriber struct {
	pub *Publisher
}

func (self *Subscriber) Streams() (streams []av.CodecData) {
	pub := self.pub
	pub.lock.RLock()
	streams = pub.streams
	pub.lock.RUnlock()
	return
}

func (self *Subscriber) ReadPacket() (i int, pkt av.Packet, err error) {
	pub := self.pub
	cond := pub.cond
	cond.L.Lock()
	ppkt := &pub.pkt
	if pub.closed {
		ppkt = nil
	} else {
		cond.Wait()
	}
	cond.L.Unlock()

	if ppkt == nil {
		err = io.EOF
		return
	}
	i, pkt = ppkt.i, ppkt.Packet
	return
}

func (self *Subscriber) Close() (err error) {
	if self.pub != nil {
		self.pub.removeSubscriber()
		self.pub = nil
	}
	return
}

type Context struct {
	publishers  map[uint64]*Publisher
	lock        *sync.RWMutex
	onSubscribe func(*Publisher)
}

func New() *Context {
	context := &Context{}
	context.lock = &sync.RWMutex{}
	context.publishers = make(map[uint64]*Publisher)
	return context
}

func (self *Context) HandleSubscribe(fn func(*Publisher)) {
	self.onSubscribe = fn
}

func (self *Context) Publish(params ...interface{}) (pub *Publisher, err error) {
	var h uint64
	if h, err = hashParams(params); err != nil {
		err = fmt.Errorf("please use string/int in Publish() params")
		return
	}

	self.lock.Lock()
	pub, exists := self.publishers[h]
	if !exists {
		pub = newPublisher()
		pub.Params = params
		pub.h = h
		pub.context = self
		self.publishers[h] = pub
	}
	self.lock.Unlock()

	if exists {
		err = fmt.Errorf("publisher already exist")
		return
	}

	return
}

func (self *Context) Subscribe(params ...interface{}) (sub *Subscriber, err error) {
	var h uint64
	if h, err = hashParams(params); err != nil {
		err = fmt.Errorf("please use string/int in Subscribe() params")
		return
	}
	needcb := false

	self.lock.RLock()
	pub := self.publishers[h]
	if pub == nil && self.onSubscribe != nil {
		pub = newPublisher()
		pub.Params = params
		pub.h = h
		pub.context = self
		self.publishers[h] = pub
		pub.ondemand = true
		needcb = true
	}
	self.lock.RUnlock()

	if pub == nil {
		err = fmt.Errorf("publisher not found")
		return
	}

	if needcb {
		go func() {
			self.onSubscribe(pub)
			pub.Close()
		}()
	}

	if sub, err = pub.addSubscriber(); err != nil {
		return
	}

	return
}
