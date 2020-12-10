package main

import (
	"errors"
	"github.com/kazhmir/gna"
	"github.com/kazhmir/gna/examples/blobs/shared"
	"log"
	"sync"
	"time"
)

func main() {
	server := Server{
		make(map[uint64]*shared.Blob, 64),
		500,
		&EventList{list: make([]*shared.Event, 128)},
		sync.Mutex{},
	}
	gna.SetStdTimeout(5 * time.Second)
	gna.SetStdTPS(20)
	ins := gna.NewInstance(&server)
	if err := gna.RunServer("0.0.0.0:8888", ins); err != nil {
		log.Fatal(err)
	}
}

type Server struct {
	blobs map[uint64]*shared.Blob
	size  int
	evl   *EventList
	mu    sync.Mutex
}

func (sr *Server) Update(ins *gna.Instance) {
	sr.mu.Lock()
	defer sr.mu.Unlock()
	ins.Broadcast(sr.Events()...)
	data := ins.GetData()
	for _, input := range data {
		v, _ := input.Data.(string)
		b := sr.blobs[input.P.ID]
		for j := range v {
			switch v[j] {
			case 'w':
				b.Move(false)
			case 's':
				b.Move(true)
			case 'a':
				b.Rotate(false)
			case 'd':
				b.Rotate(true)
			}
		}
		ins.Broadcast(b)
	}
}

func (sr *Server) Disconn(ins *gna.Instance, p *gna.Player) {
	sr.RmBlob(p.ID)
}

func (sr *Server) Auth(ins *gna.Instance, p *gna.Player) {
	a, err := p.Recv()
	if err != nil {
		log.Println(err)
		p.Terminate()
	}
	if auth, ok := a.(shared.Auth); ok {
		if auth.Pwd == "password" {
			p.Send(sr.NewBlob(p.ID))
			return
		}
	}
	p.Send(errors.New("invalid password"))
	p.Terminate()
}

func (gm *Server) NewBlob(id uint64) *shared.Blob {
	b := &shared.Blob{}
	gm.mu.Lock()
	gm.blobs[id] = b
	gm.evl.Add(&shared.Event{ID: id, T: shared.EBorn})
	gm.mu.Unlock()
	return b.Spawn(gm.size)
}

func (gm *Server) RmBlob(id uint64) {
	gm.mu.Lock()
	delete(gm.blobs, id)
	gm.evl.Add(&shared.Event{ID: id, T: shared.EDied})
	gm.mu.Unlock()
}

func (gm *Server) Events() []interface{} {
	return gm.evl.Consume()
}

type EventList struct {
	list []*shared.Event
	i    int
}

func (evl *EventList) Add(e *shared.Event) {
	if evl.i >= len(evl.list) {
		evl.list = append(evl.list, make([]*shared.Event, 64)...)
	}
	evl.list[evl.i] = e
	evl.i++
}

func (evl *EventList) Consume() []interface{} {
	out := make([]interface{}, evl.i)
	for i := 0; i < evl.i; i++ {
		out[i] = evl.list[i]
	}
	evl.i = 0
	return out
}
