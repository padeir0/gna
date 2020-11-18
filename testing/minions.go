package main

import (
	"encoding/binary"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

var host *string
var read *bool
var loop *bool
var interval time.Duration
var keepAlive time.Duration
var latency time.Duration

var (
	ErrBadPacketSize = errors.New("packet size specified by server was bigger than total bytes read")
)

type packet struct {
	size int
	data string
}

func main() {
	between := flag.Int("i", 20, "Interval between packets in milliseconds")
	minions := flag.Int("m", 1, "Number of minions")
	read = flag.Bool("r", false, "Print server response.")
	host = flag.String("h", "localhost:8888", "Host address")
	noClose := flag.Int("alive", 0, "Keep connection alive for specified seconds after sending packages")
	loop = flag.Bool("l", false, "Loops infinetely while sending packets.")
	ping := flag.Int("ping", 0, "Client Ping in milliseconds")
	flag.Parse()
	interval = time.Millisecond * time.Duration(*between)
	keepAlive = time.Second * time.Duration(*noClose)
	latency = time.Millisecond * time.Duration(*ping)
	args := flag.Args()
	if len(args) < 1 {
		fmt.Println("Fuck you")
		return
	}

	data := make([]packet, len(args))
	for i := 0; i < len(args); i++ {
		lst := strings.Split(args[i], ":")
		pktSize, err := strconv.Atoi(lst[0])
		Error(err)
		data[i] = packet{pktSize, lst[1]}
	}
	var wg sync.WaitGroup
	for i := 0; i < *minions; i++ {
		go func() {
			defer wg.Done()
			(&minion{}).start(data)
		}()
		wg.Add(1)
	}
	wg.Wait()
}

type minion struct {
	conn net.Conn
	buff []byte
	i, n int
}

func (m *minion) start(data []packet) {
	var err error
	m.conn, err = net.Dial("tcp", *host)
	Error(err)
	defer m.conn.Close()
	var n int
	var nOfPkts int
	m.buff = make([]byte, 64)

	for {
		for i := range data {
			go func() {
				n, err = sendPacket(m.conn, data[i].size, data[i].data)
				Error(err)
				fmt.Println(n, "bytes written.")
				nOfPkts++
			}()
			time.Sleep(interval)
		}
		if *read {
			for nOfPkts > 0 {
				tp, b, err := m.getPack()
				if err == io.EOF {
					return
				}
				Error(err)
				fmt.Printf("Recv: %v, Type: %v\n", b, tp)
				nOfPkts--
			}
		}
		nOfPkts = 0
		if *loop {
			time.Sleep(interval)
			continue
		}
		break
	}
	time.Sleep(keepAlive)
}

func sendPacket(conn net.Conn, size int, data string) (int, error) {
	header := make([]byte, 3)
	binary.BigEndian.PutUint16(header, uint16(size))
	header[2] = 2
	buff := append(header, []byte(data)...)
	time.Sleep(latency)
	n, err := conn.Write(buff)
	fmt.Println("Sent: ", buff[:n])
	return n, err
}

func Error(err error) {
	if err != nil {
		panic(err)
	}
}

func (t *minion) getPack() (byte, []byte, error) {
	err := t.fillBuffer(3)
	if err != nil {
		return 0, nil, err
	}
	size := binary.BigEndian.Uint16(t.buff[t.i : t.i+2])
	t.i += 2
	tp := t.buff[t.i] // type
	t.i++
	err = t.fillBuffer(int(size))
	if err != nil {
		return 0, nil, err
	}
	oldI := t.i
	t.i += int(size)
	return tp, t.buff[oldI:t.i], nil
}

func (t *minion) fillBuffer(size int) error {
	if size > len(t.buff) {
		t.buff = append(t.buff, make([]byte, size-len(t.buff))...)
	}
	if remainder := t.n - t.i; remainder < size {
		for i := 0; i < remainder; i++ { // copy end of buffer to the beginning
			t.buff[i] = t.buff[t.i+i]
		}
		if remainder < 0 || remainder > len(t.buff) {
			t.i = 0
			t.n = 0
			return ErrBadPacketSize
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
