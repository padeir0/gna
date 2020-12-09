package mgs

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

/*Sender permits the dispatching of responses through the Dispatcher.
It is implemented to reduce the time waiting for Syscalls to a minimum.*/
type Sender interface {
	/*send sends data to the destination.
	The send method should not halt, its only job is to send
	the data to a different goroutine.*/
	send(data []interface{})
	/*rectify receives a pointer to a map with the current available talkers and
	returns if the dispatcher should proceed to the Send method. If
	a implementation is independent of talkers, it should just ignore the
	received pointer and return true. It is not safe to write to the map, only to read.
	*/
	rectify(currPlayers *map[uint64]*Player) (send bool)
}

/*Input is a simple struct that contains
the data sent from the talker and a pointer to the talker.
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

type dispatcher struct {
	p  map[uint64]*Player
	d  map[Sender][]interface{}
	mu sync.Mutex
}

func (dp *dispatcher) addPlayer(p *Player) {
	dp.mu.Lock()
	dp.p[p.ID] = p
	dp.mu.Unlock()
}

func (dp *dispatcher) rmPlayer(id uint64) {
	dp.mu.Lock()
	defer dp.mu.Unlock()
	delete(dp.p, id)
}

func (dp *dispatcher) killAll() { // this will probable go bad
	for _, p := range dp.p {
		p.Terminate()
	}
}

func (dp *dispatcher) dispatch() {
	dp.mu.Lock()
	defer dp.mu.Unlock()
	for s, dt := range dp.d {
		if s.rectify(&dp.p) {
			s.send(dt)
		}
	}
	dp.d = make(map[Sender][]interface{}, 64)
}

func (dp *dispatcher) currPlayers() *Group {
	return &Group{
		tMap: dp.p, // in this case it doesn't matter if it's by reference or value
	}
}

func (dp *dispatcher) addDisp(s Sender, dt []interface{}) {
	dp.mu.Lock()
	if old, ok := dp.d[s]; ok {
		dt = append(old, dt...)
	}
	dp.d[s] = dt
	dp.mu.Unlock()
}
