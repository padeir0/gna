package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/kazhmir/gna"
	"github.com/kazhmir/gna/examples/manyChatRooms/client"
	"github.com/kazhmir/gna/examples/manyChatRooms/server"
	"github.com/kazhmir/gna/examples/manyChatRooms/shared"
)

var (
	addr = flag.String("server", "", "Hosting address (for you to host your own room).")
	host = flag.String("client", "", "Server address (the room you're trying to connect).")
	name = flag.String("name", "idiot", "Client name.")
)

var (
	cli  *gna.Client
	done = make(chan struct{})
	term = bufio.NewReader(os.Stdin)
)

func main() {
	flag.Parse()
	gna.Register(shared.SrAuth{}, shared.CliAuth{}, shared.Message{}, shared.Cmd{})
	if *addr == "" && *host == "" {
		fmt.Println("You need to either host or connect. Use '-serve <addr>' to host or '-conn <addr>' to connect.")
		return
	}
	if *addr != "" {
		go func() {
			if err := server.Start(*addr); err != nil {
				log.Fatal(err)
			}
			close(done)
		}()
	} else {
		close(done)
	}
	if *host != "" {
		fmt.Printf("Dialing %v as '%v'.\n", *host, *name)
		cli, err := client.NewC(*host, *name, term)
		if err != nil {
			log.Fatal(err)
		}
		cli.Start()
	}
	fmt.Println("beep")
	<-done
	fmt.Println("Exited.")
}
