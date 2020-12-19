package main

import (
	"github.com/kazhmir/gna"
	"log"
	"sync"
)

type Room struct {
	Users map[uint64]string
	mu    sync.Mutex
	gna.Net
}

func (r *Room) Update() {
	dt := r.GetData()
	for _, input := range dt {
		if s, ok := input.Data.(string); ok {
			r.mu.Lock()
			r.Dispatch(r.Players, Message{Username: r.Users[input.P.ID], Data: s})
			r.mu.Unlock()
		}
	}
}

func (r *Room) Auth(p *gna.Player) {
	dt, err := p.Recv()
	if err != nil {
		log.Println(err)
		p.Close()
		return
	}
	if v, ok := dt.(cliAuth); ok {
		r.mu.Lock()
		r.Users[p.ID] = v.Name
		log.Printf("%v (ID: %v) Connected.\n", v.Name, p.ID)
		r.Dispatch(r.Players, Message{Username: "server", Data: v.Name + " Connected."})
		r.mu.Unlock()
	}
	r.mu.Lock()
	err = p.Send(srAuth{UserID: p.ID})
	r.mu.Unlock()
	if err != nil {
		log.Println(err)
		p.Close()
	}
}

func (r *Room) Disconn(p *gna.Player) {
	log.Printf("%v (ID: %v) Disconnected. Reason: %v\n", r.Users[p.ID], p.ID, p.Error())
	r.Dispatch(r.Players, Message{Username: "server", Data: r.Users[p.ID] + " Disconnected."})
	r.mu.Lock()
	delete(r.Users, p.ID)
	r.mu.Unlock()
}
