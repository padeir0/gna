package main

import (
	"encoding/binary"
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
				b, err := m.getPack()
				if err == io.EOF {
					return
				}
				Error(err)
				fmt.Println("Recv: ", b)
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
	bSize := make([]byte, 2)
	binary.BigEndian.PutUint16(bSize, uint16(size))
	buff := append(bSize, []byte(data)...)
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

func (m *minion) getPack() ([]byte, error) {
	err := m.fillBuffer(2)
	if err != nil {
		return nil, err
	}
	size := binary.BigEndian.Uint16(m.buff[m.i : m.i+2])
	m.i += 2
	err = m.fillBuffer(int(size))
	if err != nil {
		return nil, err
	}
	oldI := m.i
	m.i += int(size)
	return m.buff[oldI:m.i], nil
}

func (m *minion) fillBuffer(size int) error {
	if size > len(m.buff) {
		m.buff = append(m.buff, make([]byte, size-len(m.buff))...)
	}
	if remainder := m.n - m.i; remainder < size {
		for i := m.i; i < m.n; i++ { // copy end of buffer to the beginning
			m.buff[i-m.i] = m.buff[m.i+i]
		}
		n, err := m.conn.Read(m.buff[remainder:])
		m.n = n + remainder
		m.i = 0
		if err != nil {
			return err
		}
	}
	return nil
}
