package mgs

import (
	"fmt"
	"io"
	"net"
	"time"
)

type Input struct {
	t    *talker
	time int64
	data interface{}
}

func (i *Input) String() string {
	return fmt.Sprintf("{%v %v %v}", i.t, i.time, i.data)
}

type talker struct {
	id      int
	conn    *net.TCPConn
	castOut chan io.WriterTo
	sr      *Server
}

func (t *talker) Terminate() {
	t.conn.Close()
	t.sr.signal <- t.id
	/* // should be on caster
	if *v {
		fmt.Println(t.id, "Terminated")
	}
	*/
}

func (t *talker) Talk() {
	defer t.Terminate()
	size := make([]byte, 1) // 1 byte of size, n bytes of data, n <= 255, this must change
	buff := make([]byte, 255)
	var n int
	var err error
Loop:
	for {
		select {
		case dt, ok := <-t.castOut:
			_, err = dt.WriteTo(t.conn)
			if err != nil {
				fmt.Println(err)
			}
			if !ok {
				break Loop
			}
		default:
			n, err = t.conn.Read(size)
			if err != nil {
				if opErr, ok := err.(*net.OpError); ok && opErr.Timeout() || err == io.EOF {
					break Loop
				}
				fmt.Println(err)
			}
			now := time.Now().UnixNano()
			n, err = t.conn.Read(buff[:size[0]])
			if err != nil {
				fmt.Println(err)
				break Loop
			}
			t.sr.talkIn <- &Input{t, now, t.sr.Protocol(buff[:n])}
		}
	}
}
