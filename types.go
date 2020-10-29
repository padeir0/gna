package mgs

import (
	"encoding"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"time"
)

type Input struct {
	T    *talker
	Time int64
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
	castOut chan io.WriterTo
	sr      *Server
}

func (t *talker) Terminate() {
	t.Conn.Close()
	t.sr.signal <- t.Id
}

/*Separate talker into a setup and listener,
Instead of io.WriterTo use Marshaler and append packets with lenght
Should time be appended to the packet automagically?
- Can calculate ping this way, but takes 4 bytes
*/
func (t *talker) Talk() {
	defer t.Terminate()
	bSize := make([]byte, 2) // 1 byte of size, n bytes of data, n <= 255, this must change
	buff := make([]byte, 255)
	n, err := getPack(t.Conn, &bSize, &buff)
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
	for {
		select {
		case dt, ok := <-t.castOut:
			_, err = dt.WriteTo(t.Conn)
			if t.sr.Verbose {
				dt.WriteTo(os.Stdout)
			}
			if err != nil {
				log.Print(err)
				return
			}
			if !ok {
				return
			}
		default:
			n, err = getPack(t.Conn, &bSize, &buff)
			if err != nil {
				if opErr, ok := err.(*net.OpError); ok && opErr.Timeout() || err == io.EOF {
					return
				}
				fmt.Println(err)
				continue
			}
			now := time.Now().UnixNano()
			t.sr.talkIn <- &Input{t, now, t.sr.Unmarshaler(buff[:n])}
		}
	}
}

func getPack(conn *net.TCPConn, bSize, buff *[]byte) (int, error) {
	var size uint16
	var err error
	var n int
	n, err = conn.Read(*bSize)
	size = binary.BigEndian.Uint16(*bSize)
	if err != nil {
		return n, err
	}
	var i uint16
	for i < size {
		if int(i) >= len(*buff) {
			(*buff) = append(*buff, make([]byte, 32)...)
		}
		n, err = conn.Read((*buff)[i:size])
		if err != nil {
			return n, err
		}
		i += uint16(n)
	}
	return int(size), nil
}
