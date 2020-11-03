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
}

func (t *talker) Terminate() {
	t.conn.Close()
	t.sr.signal <- t.Id
}

func (t *talker) start() {
	defer t.Terminate()
	bSize := make([]byte, 2) // uint16
	bTime := make([]byte, 8) // uint64
	buff := make([]byte, 255)
	n, _, err := getPack(t.conn, bSize, bTime, &buff)
	if err != nil && !(err == io.EOF && n > 0) {
		log.Println(err)
		if opErr, ok := err.(*net.OpError); ok && opErr.Timeout() {
			return
		}
		return
	}

	if pkt, ok := t.sr.Validate(buff[:n]); !ok {
		return
	} else {
		err = WriteTo(bSize, t.conn, pkt)
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
	var now uint64
	for {
		err = t.conn.SetReadDeadline(time.Now().Add(t.sr.Timeout))
		if err != nil {
			log.Print(err)
		}
		n, now, err = getPack(t.conn, bSize, bTime, &buff)
		// theres something wrong with this error handling
		if err != nil && !(err == io.EOF && n > 0) {
			log.Println(err)
			if opErr, ok := err.(*net.OpError); ok && opErr.Timeout() {
				return
			}
			return
		}
		t.Ping = time.Now().Sub(time.Unix(0, int64(now)))
		t.sr.talkIn <- &Input{t, now, t.sr.Unmarshaler(buff[:n])}
	}
}

func getPack(conn *net.TCPConn, bSize, bTime []byte, buff *[]byte) (int, uint64, error) {
	var time uint64
	var size uint16
	var err error
	var n int
	n, err = conn.Read(bSize)
	size = binary.BigEndian.Uint16(bSize)
	if err != nil {
		return int(size), 0, err
	}
	n, err = conn.Read(bTime)
	time = binary.BigEndian.Uint64(bTime)
	if err != nil {
		return int(size), time, err
	}
	if int(size) > len(*buff)-1 {
		*buff = append(*buff, make([]byte, int(size)-len(*buff))...)
	}

	var i uint16
	for i < size {
		n, err = conn.Read((*buff)[i:size])
		if err != nil {
			return n, 0, err
		}
		i += uint16(n)
	}
	return int(size), time, nil
}
