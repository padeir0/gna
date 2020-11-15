package mgs

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"syscall"
	"time"
)

type Encoder interface {
	Size() int
	Encode([]byte) error
}

type Sender interface {
	Send(chan struct{}, []Encoder)
	Retify(*map[uint64]*talker) bool
}

type Input struct {
	T    *talker
	Data interface{}
}

func (i *Input) String() string {
	return fmt.Sprintf("{%v %v}", i.T, i.Data)
}

func (i *Input) Size() int {
	if v, ok := i.Data.(Encoder); ok {
		return 8 + v.Size()
	}
	return 8
}

func (i *Input) Encode(buff []byte) error {
	var err error
	binary.BigEndian.PutUint64(buff, i.T.Id)
	if v, ok := i.Data.(Encoder); ok {
		err = v.Encode(buff[8:])
	}
	return err
}

type talker struct {
	Id   uint64
	conn *net.TCPConn
	sr   *Server

	mouthSig chan chan struct{}
	mouthDt  chan []Encoder

	buff []byte
	i, n int

	dead bool
	mu   sync.Mutex
}

func (t *talker) Retify(mp *map[uint64]*talker) bool {
	if _, ok := (*mp)[t.Id]; !ok {
		return false
	}
	return true
}

func (t *talker) Send(sig chan struct{}, enc []Encoder) {
	t.mouthSig <- sig
	t.mouthDt <- enc
}

func (t *talker) Terminate() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if !t.dead {
		t.dead = true
		t.conn.Close()
		t.sr.signal <- t.Id // this transfers the execution to the dispatcher, deleting the talker
		close(t.mouthSig)   // so the dispatcher knows to not send data to these channels
		close(t.mouthDt)    // and it's safe to close them
		t.sr.Disconnection(int(t.Id))
	}
}

func (t *talker) start() {
	defer t.Terminate()
	t.buff = make([]byte, 1024)
	t.mouthSig = make(chan chan struct{})
	t.mouthDt = make(chan []Encoder)
	dt, err := t.getPack()
	if err != nil {
		log.Println(err)
		if opErr, ok := err.(*net.OpError); ok && opErr.Timeout() {
			return
		}
		return
	}

	if pkt, ok := t.sr.Validate(int(t.Id), dt); !ok {
		return
	} else {
		err = WriteTo(t.conn, pkt)
		if err != nil {
			log.Print(err)
			return
		}
	}
	c := make(chan struct{})
	go func() {
		defer func() {c<- struct{}{}}()
		t.mouth()
	}()
	go func() {
		defer func() {c<- struct{}{}}()
		t.ear()
	}()
	<-c
}

func (t *talker) mouth() {
	dt := []Encoder{}
	buff := make([]byte, 256)
	for {
		sig, ok := <-t.mouthSig
		if sig == nil || !ok {
			return
		}
		dt = <-t.mouthDt
		<-sig
		n := 0
		for i := range dt {
			size := dt[i].Size()
			if n+size >= len(buff) {
				buff = append(buff, make([]byte, 2+size)...)
			}
			binary.BigEndian.PutUint16(buff[n:], uint16(size))
			n += 2
			err := dt[i].Encode(buff[n:])
			n += size
			if err != nil {
				log.Println(err)
				return
			}
		}
		x := 0
		for x < n {
			i, err := t.conn.Write(buff[x:n])
			x += i
			if err != nil {
				if errors.Is(err, syscall.EPIPE) {
					log.Println("Cancelling packets to: ", t.Id, ".", err)
					return
				}
				log.Println(err)
				break
			}
		}
	}
}

func WriteTo(w io.Writer, enc Encoder) error {
	size := enc.Size()
	buff := make([]byte, 2+size)
	binary.BigEndian.PutUint16(buff, uint16(size))
	err := enc.Encode(buff[2:])
	if err != nil {
		log.Println(err)
		return err
	}
	_, err = w.Write(buff)
	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}

func (t *talker) ear() {
	for {
		err := t.conn.SetReadDeadline(time.Now().Add(t.sr.Timeout))
		if err != nil {
			log.Println(err)
		}
		dt, err := t.getPack()

		// this error handling is still not good enough
		if err != nil {
			log.Println(err)
			if opErr, ok := err.(*net.OpError); ok && opErr.Timeout() {
				return
			}
			return
		}
		t.sr.talkIn <- &Input{t, t.sr.Unmarshaler(dt)}
	}
}

func (t *talker) getPack() ([]byte, error) {
	err := t.fillBuffer(2)
	if err != nil {
		return nil, err
	}
	size := binary.BigEndian.Uint16(t.buff[t.i : t.i+2])
	t.i += 2
	err = t.fillBuffer(int(size))
	if err != nil {
		return nil, err
	}
	oldI := t.i
	t.i += int(size)
	return t.buff[oldI:t.i], nil
}

func (t *talker) fillBuffer(size int) error {
	if size > len(t.buff) {
		t.buff = append(t.buff, make([]byte, size-len(t.buff))...)
	}
	if remainder := t.n - t.i; remainder < size {
		for i := t.i; i < t.n; i++ { // copy end of buffer to the beginning
			t.buff[i-t.i] = t.buff[t.i+i]
		}
		n, err := t.conn.Read(t.buff[remainder:])
		t.n = n + remainder
		t.i = 0
		if err != nil {
			return err
		}
	}
	return nil
}

type Group struct {
	tMap map[uint64]*talker
	mu   sync.Mutex
}

func (g *Group) Add(t *talker) {
	g.mu.Lock()
	g.tMap[t.Id] = t
	g.mu.Unlock()
}

func (g *Group) Rm(id uint64) {
	g.mu.Lock()
	delete(g.tMap, id)
	g.mu.Unlock()
}

func (g *Group) Retify(mp *map[uint64]*talker) bool {
	for id := range *mp {
		if _, ok := g.tMap[id]; !ok {
			g.Rm(id)
		}
	}
	if len(g.tMap) == 0 {
		return false
	}
	return true
}

func (g *Group) Send(sig chan struct{}, enc []Encoder) {
	for _, t := range g.tMap {
		t.mouthSig <- sig
		t.mouthDt <- enc
	}
}
