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
	gna.Register(shared.Blob{}, shared.Point{}, shared.Event{})
	server := Server{
		make(map[uint64]*shared.Blob, 64),
		500,
		sync.Mutex{},
	}
	gna.SetStdTimeout(60 * time.Second)
	gna.SetStdTPS(20)
	ins := gna.NewInstance(&server)
	if err := gna.RunServer("0.0.0.0:8888", ins); err != nil {
		log.Fatal(err)
	}
}

type Server struct {
	blobs map[uint64]*shared.Blob
	size  int
	mu    sync.Mutex
}

func (sr *Server) Update(ins *gna.Instance) {
	sr.mu.Lock()
	defer sr.mu.Unlock()
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
	ins.Broadcast(shared.Event{ID: p.ID, T: shared.EDied})
	log.Printf("%v Disconnected. Reason: %v\n", p.ID, p.Error())
}

func (sr *Server) Auth(ins *gna.Instance, p *gna.Player) {
	log.Println(p.ID, "is trying to connect!")
	a, err := p.Recv()
	if err != nil {
		log.Println(err)
		p.Terminate()
	}
	if pwd, ok := a.(string); ok {
		if pwd == "password" {
			ins.Broadcast(shared.Event{ID: p.ID, T: shared.EBorn})
			p.Send(sr.NewBlob(p.ID))
			return
		}
	}
	p.Send(errors.New("invalid password"))
	p.Terminate()
}

func (gm *Server) NewBlob(id uint64) *shared.Blob {
	b := &shared.Blob{ID: id}
	gm.mu.Lock()
	gm.blobs[id] = b
	gm.mu.Unlock()
	return b.Spawn(gm.size)
}
