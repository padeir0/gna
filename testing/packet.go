package main

import (
	"encoding/binary"
	"flag"
	"fmt"
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

type packet struct {
	size int
	data string
}

func main() {
	host = flag.String("h", "localhost:8888", "Host address")
	noClose := flag.Int("a", 0, "Keep connection alive for specified seconds after sending packages")
	between := flag.Int("i", 0, "Interval between packets in milliseconds")
	minions := flag.Int("m", 1, "Number of minions")
	frenzy := flag.Int("s", 0, "Intervals between minion-frenzys, in milliseconds. If zero theres no frenzy")
	read = flag.Bool("r", false, "Print server response.")
	flag.Parse()
	interval = time.Millisecond * time.Duration(*between)
	keepAlive = time.Second * time.Duration(*noClose)
	frenzyInterval := time.Millisecond * time.Duration(*frenzy)
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
	var n int
	var taim uint64
	buff := make([]byte, 255)
	bSize := make([]byte, 2)
	bTime := make([]byte, 8)

	defer conn.Close()
	for {
		for i := range data {
			n, err = sendPacket(conn, data[i].size, data[i].data)
			Error(err)
			fmt.Println(n, "bytes written.")
			time.Sleep(interval)
		}
		if *read {
			for {
				n, taim, err = getPack(conn, bSize, bTime, buff)
				if n <= 0 {
					break
				}
				Error(err)
				fmt.Println("Recv: ", taim, buff[:n])
			}
		}
		if t > 0 {
			time.Sleep(t)
		} else {
			break
		}
	}
	time.Sleep(keepAlive)
}

func sendPacket(conn net.Conn, size int, data string) (int, error) {
	sizeAndtime := make([]byte, 10)
	binary.BigEndian.PutUint16(sizeAndtime, uint16(size))
	binary.BigEndian.PutUint64(sizeAndtime[2:], uint64(time.Now().UnixNano()))
	buff := append(sizeAndtime, []byte(data)...)
	n, err := conn.Write(buff)
	fmt.Println("Sent: ", buff[:n])
	return n, err
}

func getPack(conn net.Conn, bSize, bTime, buff []byte) (int, uint64, error) {
	var time uint64
	var size uint16
	var err error
	var n int
	n, err = conn.Read(bSize)
	size = binary.BigEndian.Uint16(bSize)
	if err != nil {
		return n, 0, err
	}
	n, err = conn.Read(bTime)
	time = binary.BigEndian.Uint64(bTime)
	fmt.Println("Size/Time: ", size, time)
	if err != nil {
		return n, 0, err
	}
	if int(size) > len(buff)-1 {
		buff = append(buff, make([]byte, int(size)-len(buff))...)
	}

	var i uint16
	for i < size {
		n, err = conn.Read(buff[i:size])
		if err != nil {
			return n, 0, err
		}
		i += uint16(n)
	}
	return int(size), time, nil
}

func Error(err error) {
	if err != nil {
		panic(err)
	}
}
