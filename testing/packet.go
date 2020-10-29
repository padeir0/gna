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
	if *frenzy > 0 {
		for {
			for i := 0; i < *minions; i++ {
				go minion(data) // LET IT LEAK
			}
			time.Sleep(frenzyInterval)
		}
	} else {
		var wg sync.WaitGroup
		for i := 0; i < *minions; i++ {
			go func() {
				defer wg.Done()
				minion(data)
			}()
			wg.Add(1)
		}
		wg.Wait()
	}
}

func minion(data []packet) {
	conn, err := net.Dial("tcp", *host)
	Error(err)
	defer conn.Close()
	for i := range data {
		n, err := sendPacket(conn, data[i].size, data[i].data)
		Error(err)
		fmt.Println(n, "bytes written.")
		time.Sleep(interval)
	}
	if *read {
		var n int
		var err error
		buff := make([]byte, 255)
		//bSize := make([]byte, 2)
		for {
			//n, err = getPack(conn, &bSize, &buff)
			n, err = conn.Read(buff)
			if n <= 0 {
				break
			}
			Error(err)
			fmt.Println(string(buff[:n]))
		}
	}
	time.Sleep(keepAlive)
}

func sendPacket(conn net.Conn, size int, data string) (int, error) {
	packet := []byte{}
	bSize := make([]byte, 2)
	binary.BigEndian.PutUint16(bSize, uint16(size))
	buff := append(bSize, []byte(data)...)
	packet = append(packet, buff...)
	n, err := conn.Write(packet)
	fmt.Println(packet)
	return n, err
}

func getPack(conn net.Conn, bSize, buff *[]byte) (int, error) {
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
		n, err = conn.Read((*buff)[i : size-i])
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
