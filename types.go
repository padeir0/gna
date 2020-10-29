package mgs

import (
	"encoding"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"time"
)

type Input struct {
	T    *talker
	Time uint32
	Data encoding.BinaryMarshaler
}

func (i *Input) String() string {
	return fmt.Sprintf("{%v %v %v}", i.T, i.Time, i.Data)
}

func (i *Input) WriteTo(w io.Writer) (int64, error) {
	buff := make([]byte, 6)
	binary.BigEndian.PutUint32(buff[2:], uint32(i.Time)) // hopefully time.UnixNano doesn't give negative numbers
	d, err := i.Data.MarshalBinary()
	if err != nil {
		panic(err)
	}
	binary.BigEndian.PutUint16(buff, uint16(len(d)+4))
	n, err := w.Write(append(buff, d...))
	return int64(n), err
}

type talker struct {
	Id      int
	Conn    *net.TCPConn
	Ping    time.Duration
	castOut chan encoding.BinaryMarshaler
	sr      *Server
}

func (t *talker) Terminate() {
	t.Conn.Close()
	t.sr.signal <- t.Id
}

/*
Instead of io.WriterTo use Marshaler and append packets with lenght
Should time be appended to the packet automagically?
- Can calculate ping this way, but takes 4 bytes
*/
func (t *talker) start() {
	defer t.Terminate()
	bSize := make([]byte, 2)
	bTime := make([]byte, 4)
	buff := make([]byte, 255)
	n, _, err := getPack(t.Conn, bSize, bTime, buff)
	if err != nil {
		if opErr, ok := err.(*net.OpError); ok && opErr.Timeout() || err == io.EOF {
			return
		}
		fmt.Println(err)
		return
	}

	if pkg, ok := t.sr.Validate(buff[:n]); !ok {
		return
	} else {
		_, err = pkg.WriteTo(t.Conn)
		if err != nil {
			log.Print(err)
			return
		}
	}
	t.talk(bSize, bTime, buff)
}

func (t *talker) talk(bSize, bTime, buff []byte) {
	var err error
	var n int
	var now uint32
	for {
		select {
		case dt, ok := <-t.castOut:
			b, err := dt.MarshalBinary()
			if err != nil {
				log.Print(err)
				return
			}
			binary.BigEndian.PutUint16(bSize, uint16(len(b)))
			_, err = t.Conn.Write(append(bSize, b...))
			if err != nil {
				log.Print(err)
				return
			}
			if !ok {
				return
			}
		default:
			n, now, err = getPack(t.Conn, bSize, bTime, buff)
			if err != nil {
				if opErr, ok := err.(*net.OpError); ok && opErr.Timeout() || err == io.EOF {
					return
				}
				fmt.Println(err)
				continue
			}
			t.Ping = time.Now().Sub(time.Unix(0, int64(now)))
			t.sr.talkIn <- &Input{t, now, t.sr.Unmarshaler(buff[:n])}
		}
	}
}

func getPack(conn *net.TCPConn, bSize, bTime, buff []byte) (int, uint32, error) {
	var time uint32
	var size uint16
	var err error
	var n int
	n, err = conn.Read(bSize)
	size = binary.BigEndian.Uint16(bSize)
	if err != nil {
		return n, 0, err
	}
	n, err = conn.Read(bTime)
	time = binary.BigEndian.Uint32(bTime)
	if err != nil {
		return n, 0, err
	}

	var i uint16
	for i < size {
		if int(i) >= len(buff) {
			buff = append(buff, make([]byte, 32)...)
		}
		n, err = conn.Read(buff[i:size])
		if err != nil {
			return n, 0, err
		}
		i += uint16(n)
	}
	return int(size), time, nil
}
