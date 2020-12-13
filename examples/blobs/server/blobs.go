package main

import (
	"errors"
	"flag"
	//"fmt"
	"github.com/kazhmir/gna"
	"github.com/kazhmir/gna/examples/blobs/shared"
	"log"
	"os"
	"runtime/pprof"
	"sync"
	"time"
)

var cpuprofile = flag.String("ppcpu", "", "write cpu profile to `file`")

func main() {
	flag.Parse()
	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal("could not create CPU profile: ", err)
		}
		defer f.Close() // error handling omitted for example
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatal("could not start CPU profile: ", err)
		}
		defer pprof.StopCPUProfile()
	}
	gna.Register(shared.Blob{}, shared.Point{}, shared.Event{}, []*shared.Blob{})
	server := Server{
		blobs: make(map[uint64]*shared.Blob, 64),
	}
	gna.SetReadTimeout(60 * time.Second)
	gna.SetWriteTimeout(15 * time.Second)
	gna.SetMaxTPS(20)
	ins := gna.NewInstance(&server)
	if err := gna.RunServer("0.0.0.0:8888", ins); err != nil {
		log.Fatal(err)
	}
}

type Server struct {
	blobs map[uint64]*shared.Blob
	mu    sync.Mutex
}

var i int

func (sr *Server) Update(ins *gna.Instance) {
	sr.mu.Lock()
	defer sr.mu.Unlock()
	data := ins.GetData()
	updates := []*shared.Blob{}
	for _, input := range data {
		v, _ := input.Data.(string)
		b, ok := sr.blobs[input.P.ID]
		if ok {
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
			updates = append(updates, b)
		}
	}
	/*
		fmt.Printf("%v ", len(updates))
		i++
		if i%50 == 0 {
			fmt.Println()
		}
	*/
	ins.Broadcast(updates)
}

func (sr *Server) Disconn(ins *gna.Instance, p *gna.Player) {
	ins.Broadcast(shared.Event{ID: p.ID, T: shared.EDied})
	sr.mu.Lock()
	delete(sr.blobs, p.ID)
	sr.mu.Unlock()
	log.Printf("%v Disconnected. Reason: %v\n", p.ID, p.Error())
}

func (sr *Server) Auth(ins *gna.Instance, p *gna.Player) {
	//	log.Println(p.ID, "is trying to connect!")
	a, err := p.Recv()
	if err != nil {
		log.Println(err)
		p.Close()
	}
	if pwd, ok := a.(string); ok {
		if pwd == "password" {
			b := sr.NewBlob(p.ID)
			p.Send(b)
			sr.mu.Lock()
			for _, b := range sr.blobs {
				p.Send(b)
			}
			sr.mu.Unlock()
			ins.Broadcast(shared.Event{ID: p.ID, T: shared.EBorn})
			return
		}
	}
	p.Send(errors.New("invalid password"))
	p.Close()
}

func (gm *Server) NewBlob(id uint64) *shared.Blob {
	b := &shared.Blob{ID: id}
	gm.mu.Lock()
	gm.blobs[id] = b
	gm.mu.Unlock()
	return b
}
