package gna

import (
	"encoding/gob"
	"fmt"
	"net"
	"sync"
	//"sync"
)

func newPlayer(id uint64, c net.Conn) *Player {
	return &Player{
		ID: id,
		dispatcher: dispatcher{
			conn:        c,
			rTimeout:    stdReadTimeout,
			wTimeout:    stdWriteTimeout,
			enc:         gob.NewEncoder(c),
			dec:         gob.NewDecoder(c),
			cDisp:       make(chan interface{}, 32),
			shouldStart: true,
		},
	}
}

/*Player represents the player connection,
it has a receiver and dispatcher that run concurrently above
a persistent TCP connection*/
type Player struct {
	ID  uint64
	dc  chan *Player  // chan to the disconnection handler
	acu *playerBucket // player acumulator, shared with other players in the instance
	grp *Group        // instance group
	err error         // decode/read error
	dispatcher
}

func (p *Player) start() {
	go p.receiver()
	go p.work()
}

/*SetInstance removes the player from the previous instance, if any,
and sends him to another.*/
func (p *Player) SetInstance(ins Instance) {
	n := ins.NetAbs()
	n.mu.Lock()
	defer n.mu.Unlock()
	if !n.started {
		panic("instance not started") // probably a little too harsh
	}
	if p.grp != nil {
		p.grp.Rm(p.ID)
	}
	p.grp = n.Players
	p.grp.Add(p)
	p.acu = n.acu
	p.rTimeout = n.rTimeout
	p.wTimeout = n.wTimeout
	p.dc = n.dc
}

func (p *Player) disc() {
	p.dc <- p
}

func (p *Player) receiver() {
	defer p.disc()
	defer p.Close()
	for {
		dt, err := p.Recv()
		if err != nil {
			p.err = fmt.Errorf("recv: %w", err)
			/*if opErr, ok := err.(*net.OpError); ok && opErr.Timeout() {
				return
			}*/
			return
		}
		if dt != nil { // ?
			p.acu.add(&Input{p, dt})
		}
	}
}

func (p *Player) Error() error {
	if p.err != nil {
		if p.dispatcher.err != nil {
			return fmt.Errorf("%w, alongside: %v", p.err, p.dispatcher.err)
		}
		return p.err
	}
	if p.dispatcher.err != nil {
		return p.dispatcher.err
	}
	return nil
}

type playerBucket struct {
	dt []*Input
	i  int
	mu sync.Mutex
}

func (is *playerBucket) add(dt *Input) {
	is.mu.Lock()
	if is.i >= len(is.dt) {
		is.dt = append(is.dt, make([]*Input, 64)...)
	}
	is.dt[is.i] = dt
	is.i++
	is.mu.Unlock()
}

func (is *playerBucket) consume() []*Input {
	is.mu.Lock()
	out := make([]*Input, is.i)
	copy(out, is.dt[:is.i])
	is.i = 0
	is.mu.Unlock()
	return out
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
