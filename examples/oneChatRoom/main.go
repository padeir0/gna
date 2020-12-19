package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/kazhmir/gna"
	"log"
	"os"
	"time"
)

var (
	addr   = flag.String("server", "", "Hosting address (for you to host your own room).")
	server = flag.String("client", "", "Server address (the room you're trying to connect).")
	name   = flag.String("name", "idiot", "Client name.")
)

var (
	cli  *gna.Client
	done = make(chan struct{})
	term = bufio.NewReader(os.Stdin)
)

func main() {
	flag.Parse()
	gna.Register(srAuth{}, cliAuth{}, Message{})
	if *addr == "" && *server == "" {
		fmt.Println("You need to either host or connect. Use '-serve <addr>' to host or '-conn <addr>' to connect.")
		return
	}
	if *addr != "" {
		gna.SetReadTimeout(60 * time.Second)
		gna.SetWriteTimeout(60 * time.Second)
		go func() {
			if err := gna.RunServer(*addr, &Room{Users: make(map[uint64]string, 64)}); err != nil {
				log.Fatal(err)
			}
			close(done)
		}()
	} else {
		close(done)
	}
	if *server != "" {
		fmt.Printf("Dialing %v as '%v'.\n", *server, *name)
		cli = Connect(*server, *name)
		cli.SetTimeout(60 * time.Second)
		go CliUpdate()
		ClientLoop()
	}
	<-done
	fmt.Println("Exited.")
}
