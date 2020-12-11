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

/*sender permits the dispatching of responses through the dispatcher.
It should not wait for any syscalls.*/
type sender interface {
	/*send sends data to the destination.
	The send method should not halt, its only job is to send
	the data to a different goroutine.*/
	send(data []interface{})
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

type dispatcher struct {
	d  map[sender][]interface{}
	mu sync.Mutex
}

func (dp *dispatcher) dispatch() {
	dp.mu.Lock()
	defer dp.mu.Unlock()
	if len(dp.d) == 0 {
		return
	}
	for s, dt := range dp.d {
		s.send(dt)
	}
	dp.d = make(map[sender][]interface{}, 64)
}

func (dp *dispatcher) addDisp(s sender, dt []interface{}) {
	dp.mu.Lock()
	if old, ok := dp.d[s]; ok {
		dt = append(old, dt...)
	}
	dp.d[s] = dt
	dp.mu.Unlock()
}
