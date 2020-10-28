package main

import (
	"flag"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

var host *string
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
	time.Sleep(keepAlive)
}

func sendPacket(conn net.Conn, size int, data string) (int, error) {
	packet := []byte{}
	buff := append([]byte{byte(size)}, []byte(data)...)
	packet = append(packet, buff...)
	n, err := conn.Write(packet)
	fmt.Println(packet)
	return n, err
}

func Error(err error) {
	if err != nil {
		panic(err)
	}
}
