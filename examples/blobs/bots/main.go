package main

import (
	"flag"
	"github.com/kazhmir/gna"
	"github.com/kazhmir/gna/examples/blobs/shared"
	"log"
	"math/rand"
	"sync"
	"time"
)

const (
	scrWid = 800
	scrHei = 600
)

var (
	serverAddr = flag.String("host", "localhost:8888", "Host address <ip>:<port>")
	pwd        = flag.String("pwd", "password", "Host Password")
	nBots      = flag.Int("m", 1, "Number of bots")
)

func main() {
	gna.Register(shared.Blob{}, []*shared.Blob{}, shared.Point{}, shared.Event{})
	flag.Parse()

	var wg sync.WaitGroup
	for i := 0; i < *nBots; i++ {
		client, _ := Connect(*serverAddr, *pwd)
		wg.Add(1)
		go func() {
			RunBot(client)
			wg.Done()
		}()
		time.Sleep(50 * time.Millisecond)
	}
	wg.Wait()
}

func Connect(addr, pwd string) (*gna.Client, *shared.Blob) {
	client, err := gna.Dial(addr)
	if err != nil {
		panic(err)
	}
	err = client.Send(pwd)
	if err != nil {
		panic(err)
	}
	dt, err := client.Recv()
	if err != nil {
		panic(err)
	}
	v, ok := dt.(shared.Blob)
	if ok {
		client.SetTimeout(60 * time.Second)
		client.Start()
		return client, &v
	}
	panic("data was not blob")
}

func RunBot(c *gna.Client) {
	defer c.Close()
	ticker := time.NewTicker(20 * time.Millisecond)
	for i := 0; ; i++ {
		_ = c.RecvBatch() /* explicitly discarding the recv data for the TCP buffer be freed,
		if the TCP buffer gets full, the server will send data until the channel Player.cDisp is full
		and then close the connection. It is necessary to close the connection in this case to free
		server resources in case o faulty receivers.
		*/
		<-ticker.C
		c.Send(Input(i))
		if err := c.Error(); err != nil {
			log.Println(err)
			return
		}
	}
}

var getToCenter = "dddddddddddddddddwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwwww"
var inputs = "wasd"

func Input(i int) string {
	if i < len(getToCenter) {
		return string(getToCenter[i])
	}
	rand.Seed(time.Now().UnixNano())
	return string(inputs[rand.Intn(len(inputs))])
}
