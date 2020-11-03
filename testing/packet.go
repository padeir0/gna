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
	frenzy := flag.Int("frenzy", 0, "Intervals between frenzys, in milliseconds. If zero theres no frenzy.")
	ping := flag.Int("ping", 0, "Client Ping in milliseconds")
	flag.Parse()
	interval = time.Millisecond * time.Duration(*between)
	keepAlive = time.Second * time.Duration(*noClose)
	frenzyInterval := time.Millisecond * time.Duration(*frenzy)
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
			minion(data, frenzyInterval)
		}()
		wg.Add(1)
	}
	wg.Wait()
}

func minion(data []packet, t time.Duration) {
	conn, err := net.Dial("tcp", *host)
	Error(err)
	defer conn.Close()
	var n int
	var nOfPkts int
	buff := make([]byte, 255)
	bSize := make([]byte, 2)

	for {
		for i := range data {
			n, err = sendPacket(conn, data[i].size, data[i].data)
			Error(err)
			fmt.Println(n, "bytes written.")
			nOfPkts++
			time.Sleep(interval)
		}
		if *read {
			for nOfPkts > 0 {
				n, err = getPack(conn, bSize, &buff)
				if err == io.EOF {
					return
				}
				Error(err)
				fmt.Println("Recv: ", string(buff[:n]))
				nOfPkts--
			}
		}
		if t > 0 {
			time.Sleep(t)
		} else {
			break
		}
		nOfPkts = 0
	}
	time.Sleep(keepAlive)
}

func sendPacket(conn net.Conn, size int, data string) (int, error) {
	sizeAndtime := make([]byte, 10)
	binary.BigEndian.PutUint16(sizeAndtime, uint16(size))
	binary.BigEndian.PutUint64(sizeAndtime[2:], uint64(time.Now().UnixNano()))
	buff := append(sizeAndtime, []byte(data)...)
	time.Sleep(latency)
	n, err := conn.Write(buff)
	fmt.Println("Sent: ", buff[:n])
	return n, err
}

func getPack(conn net.Conn, bSize []byte, buff *[]byte) (int, error) {
	var size uint16
	var err error
	var n int
	n, err = conn.Read(bSize)
	size = binary.BigEndian.Uint16(bSize)
	if err != nil {
		return n, err
	}
	if int(size) > len(*buff)-1 {
		*buff = append(*buff, make([]byte, int(size)-len(*buff))...)
	}

	var i uint16
	for i < size {
		n, err = conn.Read((*buff)[i:size])
		if err != nil {
			return n, err
		}
		i += uint16(n)
	}
	return int(size), nil
}

func Error(err error) {
	if err != nil {
		panic(err)
	}
}
