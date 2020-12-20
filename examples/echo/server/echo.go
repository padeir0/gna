package main

import (
	"github.com/kazhmir/gna"
	"log"
)

func main() {
	if err := gna.RunServer(":8888", &Echo{}); err != nil {
		log.Fatal(err)
	}
}

type Echo struct {
	gna.Net
}

var password = "banana"

func (e *Echo) Auth(p *gna.Player) {
	dt, err := p.Recv()
	if err != nil {
		log.Println(err)
		p.Close()
		return
	}
	if s, ok := dt.(string); ok {
		if s == password {
			log.Printf("%v Connected\n", p.ID)
			return
		}
	}
	p.Close()
}

func (e *Echo) Disconn(p *gna.Player) {
	log.Printf("%v Disconnected, Reason: %v\n", p.ID, p.Error())
}

func (e *Echo) Update() {
	dt := e.GetData()
	for _, input := range dt {
		e.Dispatch(e.Players, input.Data)
	}
}
