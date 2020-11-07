package mgs

import (
	"encoding"
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

type Input struct {
	T    *talker
	Time uint64
	Data encoding.BinaryMarshaler
}

func (i *Input) String() string {
	return fmt.Sprintf("{%v %v %v}", i.T, i.Time, i.Data)
}

func (i *Input) MarshalBinary() ([]byte, error) {
	buff := make([]byte, 12)
	binary.BigEndian.PutUint64(buff, i.Time)
	binary.BigEndian.PutUint32(buff[8:], i.T.Id)
	d, err := i.Data.MarshalBinary()
	return append(buff, d...), err
}

type talker struct {
	Id   uint32
	Ping time.Duration
	conn *net.TCPConn
	sr   *Server

	mouthSig chan chan struct{}
	mouthDt  chan []encoding.BinaryMarshaler

	buff []byte
	i, n int

	dead bool
	mu   sync.Mutex
}

func (t *talker) Terminate() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if !t.dead {
		t.conn.Close()
		t.sr.signal <- t.Id // this transfers the execution to the dispatcher, deleting the talker
		close(t.mouthSig)   // so the dispatcher knows to not send data to this channel
		close(t.mouthDt)    // and it's safe to close them
		t.dead = true
	}
}

func (t *talker) start() {
	defer t.Terminate()
	t.buff = make([]byte, 1024)
	t.mouthSig = make(chan chan struct{})
	t.mouthDt = make(chan []encoding.BinaryMarshaler)
	dt, _, err := t.getPack()
	if err != nil {
		log.Println(err)
		if opErr, ok := err.(*net.OpError); ok && opErr.Timeout() {
			return
		}
		return
	}

	if pkt, ok := t.sr.Validate(dt); !ok {
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
		defer func() { c <- struct{}{} }()
		t.mouth()
	}()
	go func() {
		defer func() { c <- struct{}{} }()
		t.ear()
	}()

	<-c // if either dies the talker must stop
}

/* Burárum this mouth is a little too big
 */
func (t *talker) mouth() {
	dt := []encoding.BinaryMarshaler{}
	buff := make([]byte, 256)
	for {
		sig := <-t.mouthSig
		if sig == nil {
			return
		}
		dt = <-t.mouthDt
		<-sig
		n := 0
		for i := range dt {
			b, err := dt[i].MarshalBinary()
			if err != nil {
				log.Println(err)
				return
			}
			size := uint16(len(b))
			binary.BigEndian.PutUint16(buff[n:], size)
			n += 2
			for j := range b {
				buff[n+j] = b[j]
			}
			n += int(size)
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

func WriteTo(w io.Writer, bm encoding.BinaryMarshaler) error {
	bSize := make([]byte, 2)
	b, err := bm.MarshalBinary()
	if err != nil {
		log.Println(err)
		return err
	}
	binary.BigEndian.PutUint16(bSize, uint16(len(b)))
	dt := append(bSize, b...)
	_, err = w.Write(dt)
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
		dt, now, err := t.getPack()

		// this error handling is still not good enough
		if err != nil {
			log.Println(err)
			if opErr, ok := err.(*net.OpError); ok && opErr.Timeout() {
				return
			}
			return
		}
		t.Ping = time.Now().Sub(time.Unix(0, int64(now)))
		t.sr.talkIn <- &Input{t, now, t.sr.Unmarshaler(dt)}
	}
}

func (t *talker) getPack() ([]byte, uint64, error) {
	err := t.fillBuffer(10)
	if err != nil {
		return nil, 0, err
	}
	size := binary.BigEndian.Uint16(t.buff[t.i : t.i+2])
	t.i += 2
	time := binary.BigEndian.Uint64(t.buff[t.i : t.i+8])
	t.i += 8
	err = t.fillBuffer(int(size))
	if err != nil {
		return nil, 0, err
	}
	oldI := t.i
	t.i += int(size)
	return t.buff[oldI:t.i], time, nil
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
