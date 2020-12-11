package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/kazhmir/gna"
	"log"
	"os"
	"strings"
	"sync"
	"time"
)

var (
	addr   = flag.String("serve", "", "Hosting address (for you to host your own room).")
	server = flag.String("conn", "", "Server address (the room you're trying to connect).")
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
		sr := &Room{Users: make(map[uint64]string, 64)}
		gna.SetStdTimeout(60 * time.Second)
		ins := gna.NewInstance(sr)
		go func() {
			if err := gna.RunServer(*addr, ins); err != nil {
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

func CliUpdate() {
	ticker := time.NewTicker(50 * time.Millisecond)
	for {
		<-ticker.C
		data, _ := cli.Recv() // err is aways nil with cli.Start()
		if err := cli.Error(); err != nil {
			log.Fatalf("Recv Error: %v\n", err)
		}
		if len(data) == 0 {
			continue
		}
		for _, dt := range data {
			switch v := dt.(type) {
			case Message:
				fmt.Printf("%v: %v\n", v.Username, v.Data)
			default:
				fmt.Printf("\n%#v, %T\n", dt, dt)
			}
		}
	}
}

func ClientLoop() {
	for {
		msg, err := term.ReadString('\n')
		if err != nil {
			fmt.Println(err)
			continue
		}
		cli.Send(strings.ReplaceAll(msg, "\n", ""))
		if err := cli.Error(); err != nil {
			log.Fatalf("Send Error: %v\n", err)
		}
	}
}

func Connect(addr, name string) *gna.Client {
	cli, err := gna.Dial(addr)
	if err != nil {
		log.Fatal(err)
	}
	err = cli.Send(cliAuth{name})
	if err != nil {
		log.Fatal(err)
	}
	dts, err := cli.Recv()
	if v, ok := dts[0].(srAuth); ok {
		fmt.Printf("UserID: %v\n", v.UserID)
	}
	cli.Start()
	return cli
}

type Room struct {
	Users map[uint64]string
	mu    sync.Mutex
}

func (r *Room) Update(ins *gna.Instance) {
	dt := ins.GetData()
	for _, input := range dt {
		if s, ok := input.Data.(string); ok {
			r.mu.Lock()
			ins.Broadcast(Message{Username: r.Users[input.P.ID], Data: s})
			r.mu.Unlock()
		}
	}
}

func (r *Room) Auth(ins *gna.Instance, p *gna.Player) {
	dt, err := p.Recv()
	if err != nil {
		log.Println(err)
		p.Terminate()
		return
	}
	if v, ok := dt.(cliAuth); ok {
		r.mu.Lock()
		r.Users[p.ID] = v.Name
		fmt.Printf("%v (ID: %v) Connected.\n", v.Name, p.ID)
		ins.Broadcast(Message{Username: "server", Data: v.Name + " Connected."})
		r.mu.Unlock()
	}
	r.mu.Lock()
	err = p.Send(srAuth{UserID: p.ID})
	r.mu.Unlock()
	if err != nil {
		log.Println(err)
		p.Terminate()
	}
}

func (r *Room) Disconn(ins *gna.Instance, p *gna.Player) {
	fmt.Printf("%v (ID: %v) Disconnected. Reason: %v\n", r.Users[p.ID], p.ID, p.Error())
	ins.Broadcast(Message{Username: "server", Data: r.Users[p.ID] + " Disconnected."})
	r.mu.Lock()
	delete(r.Users, p.ID)
	r.mu.Unlock()
}

type srAuth struct {
	UserID uint64
}
type cliAuth struct {
	Name string
}

type Message struct {
	Username string
	Data     string
}
