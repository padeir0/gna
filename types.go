package gna

import (
	"fmt"
	"sync"
)

type id struct {
	i  uint64
	mu sync.Mutex
}

func (x *id) newID() uint64 {
	var out uint64
	x.mu.Lock()
	out = x.i
	x.i++
	x.mu.Unlock()
	return out
}

type shipper interface {
	/*sends the data to the right chan for dispatching*/
	ship(interface{})
}

/*Input is a simple struct that contains
the data sent from the Player and a pointer to the Player.
*/
type Input struct {
	P    *Player
	Data interface{}
}

func (i *Input) String() string {
	return fmt.Sprintf("{%v %v}", i.P, i.Data)
}

type acumulator struct {
	dt []*Input
	i  int
	mu sync.Mutex
}

func (is *acumulator) add(dt ...*Input) {
	is.mu.Lock()
	if is.i+len(dt) > len(is.dt) {
		is.dt = append(is.dt, make([]*Input, len(dt)+64)...)
	}
	for j := range dt {
		is.dt[is.i] = dt[j]
		is.i++
	}
	is.mu.Unlock()
}

func (is *acumulator) consume() []*Input {
	is.mu.Lock()
	out := make([]*Input, is.i)
	copy(out, is.dt[:is.i])
	is.i = 0
	is.mu.Unlock()
	return out
}

type packet struct {
	s  shipper
	dt interface{}
}

func dispatcher(data chan *packet) {
	for {
		dt := <-data
		dt.s.ship(dt.dt)
	}
}

func dcHandler(dc chan *Player, ins *Instance) {
	for {
		p := <-dc
		ins.players.Rm(p.ID)
		ins.world.Disconn(ins, p)
	}
}

/*
	start := time.Now()
	end := time.Now()
	if t := end.Sub(start); t > 40*time.Millisecond {
		fmt.Printf("\n\n%v %v, %v\n\n", p.ID, t, p.Error())
	}
*/
