package gna

import (
	"time"
)

type World interface {
	Update(*Instance)
	Disconn(*Instance, *Player)
	Auth(*Instance, *Player)
}

func NewInstance(sr World) *Instance {
	return &Instance{
		world:   sr,
		timeout: stdTimeout,
		done:    make(chan struct{}),
		ticker:  time.NewTicker(time.Second / time.Duration(stdTPS)),
		players: &Group{pMap: make(map[uint64]*Player, 16)},
		acu:     &acumulator{dt: make([]*Input, 64)},
		disp:    &dispatcher{d: make(map[sender][]interface{}, 16)},
	}
}

type Instance struct { // TODO should Instance have an ID?
	world   World
	timeout time.Duration
	done    chan struct{}
	started bool
	ticker  *time.Ticker

	players *Group

	acu  *acumulator
	disp *dispatcher
}

func (ins *Instance) Start() {
	if ins.started {
		return
	}
	ins.started = true
	for {
		select {
		case <-ins.ticker.C:
			ins.world.Update(ins)
			ins.disp.dispatch()
		case <-ins.done:
			return
		}
	}
}

func (ins *Instance) terminate() {
	ins.players.Terminate()
	ins.done <- struct{}{}
}

func (ins *Instance) GetData() []*Input {
	return ins.acu.consume()
}

func (ins *Instance) Broadcast(dt ...interface{}) {
	ins.disp.addDisp(ins.players, dt)
}

func (ins *Instance) Multicast(g *Group, dt ...interface{}) {
	ins.disp.addDisp(g, dt)
}

func (ins *Instance) Unicast(p *Player, dt ...interface{}) {
	ins.disp.addDisp(p, dt)
}
