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

var (
	/*ErrBadPacketSize happens when the client sends the wrong packet size.
	One or more packets may be dropped.*/
	ErrBadPacketSize = errors.New("packet size specified by client was bigger than total bytes read")
)

/*Encoder permits the marshaling of the data structure into
binary form, it's similar to encoding.BinaryMarshaler
except it uses a given buffer to encode it, saving
on allocations.
*/
type Encoder interface {
	/*Size returns the total size of the data structure
	to be marshaled, if the size is less than what the Encode
	needs, an error or panic might ensue. (Depends on your implementation)*/
	Size() int
	/*Encode marshals the data structure into binary form, storing it into
	the given buffer and returns an error if any.*/
	Encode([]byte) error
}

/*Sender permits the dispatching of responses through the Dispatcher.
It is implemented to reduce the time waiting for Syscalls to a minimum.*/
type Sender interface {
	/*Send receives the data and a channel for signaling.
	The data and signal should be sent to separate goroutine(s), and the
	data only processed after the sig channel is closed.
	The Send method should not halt, its only job is to send
	the signal and data to a different goroutine.*/
	Send(sig chan struct{}, data []Encoder)
	/*Rectify receives a pointer to a map with the current available talkers and
	returns if the dispatcher should proceed to the Send method. If
	a implementation is independent of talkers, it should just ignore the
	received pointer and return true. It is not safe to write to the map, only to read.
	*/
	Rectify(currTalkers *map[uint64]*talker) (send bool)
}

/*Input is a simple struct that contains
the data sent from the talker and a pointer to the talker.
*/
type Input struct {
	T    *talker
	Data interface{}
}

func (i *Input) String() string {
	return fmt.Sprintf("{%v %v}", i.T, i.Data)
}

/*Size returns the size of the struct in bytes*/
func (i *Input) Size() int {
	if v, ok := i.Data.(Encoder); ok {
		return 8 + v.Size()
	}
	return 8
}

/*Encode marshals the struct into the given buffer*/
func (i *Input) Encode(buff []byte) error {
	var err error
	binary.BigEndian.PutUint64(buff, i.T.ID)
	if v, ok := i.Data.(Encoder); ok {
		err = v.Encode(buff[8:])
	}
	return err
}

type talker struct {
	/*Talker ID*/
	ID   uint64
	conn *net.TCPConn
	sr   *Server

	mouthSig chan chan struct{}
	mouthDt  chan []Encoder

	buff []byte
	i, n int

	dead bool
	mu   sync.Mutex
}

/*Rectify returns true if the talker exists in the map*/
func (t *talker) Rectify(mp *map[uint64]*talker) bool {
	if _, ok := (*mp)[t.ID]; !ok {
		return false
	}
	return true
}

/*Send sends the signal channel and data to the mouth of the talker*/
func (t *talker) Send(sig chan struct{}, enc []Encoder) {
	t.mouthSig <- sig
	t.mouthDt <- enc
}

/*Terminate terminates the talker, executing Disconnect,
closing the connection and the channels.*/
func (t *talker) Terminate() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if !t.dead {
		t.dead = true
		t.conn.Close()
		t.sr.signal <- t.ID // this transfers the execution to the dispatcher, deleting the talker
		close(t.mouthSig)   // so the dispatcher knows to not send data to these channels
		close(t.mouthDt)    // and it's safe to close them
		t.sr.Disconnection(int(t.ID))
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
	pkt, ok := t.sr.Validate(int(t.ID), dt)
	if !ok {
		return
	}
	err = writeTo(t.conn, pkt)
	if err != nil {
		log.Print(err)
		return
	}
	c := make(chan struct{})
	go func() {
		defer func() { c <- struct{}{} }()
		t.mouth()
	}()
	go func() {
		defer func() { c <- struct{}{} }()
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
					log.Println("Cancelling packets to: ", t.ID, ".", err)
					return
				}
				log.Println(err)
				break
			}
		}
	}
}

func writeTo(w io.Writer, enc Encoder) error {
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
			if err == ErrBadPacketSize {
				continue
			}
			return
		}
		unmarshaled := t.sr.Unmarshaler(dt)
		if unmarshaled != nil {
			t.sr.talkIn <- &Input{t, unmarshaled}
		}
	}
}

func (t *talker) getPack() ([]byte, error) {
	err := t.fillBuffer(2)
	if err != nil {
		return nil, err
	}
	size := binary.BigEndian.Uint16(t.buff[t.i : t.i+2])
	t.i += 2
	if int(size) > t.n-t.i {
		return nil, ErrBadPacketSize
	}
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

/*Group is a collection of talkers that is safe for concurrent use,
This can be used to "multicast" a single piece o data to a set of talkers.
*/
type Group struct {
	tMap map[uint64]*talker
	mu   sync.Mutex
}

/*Add a talker to the Group*/
func (g *Group) Add(t *talker) {
	g.mu.Lock()
	g.tMap[t.ID] = t
	g.mu.Unlock()
}

/*Rm removes a talker from the Group*/
func (g *Group) Rm(id uint64) {
	g.mu.Lock()
	delete(g.tMap, id)
	g.mu.Unlock()
}

/*Rectify removes talkers from the Group that are not in the given map*/
func (g *Group) Rectify(mp *map[uint64]*talker) bool {
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

/*Send sends the sig channel and data to each Talker in the group*/
func (g *Group) Send(sig chan struct{}, data []Encoder) {
	for _, t := range g.tMap {
		t.mouthSig <- sig
		t.mouthDt <- data
	}
}
