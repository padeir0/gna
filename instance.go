package gna

import (
	"time"
)

func NewInstance(sr Server) *Instance {
	return &Instance{
		handler: sr,
		disp:    &dispatcher{p: make(map[uint64]*Player, 16), d: make(map[sender][]interface{}, 16)},
		timeout: stdTimeout,
		done:    make(chan struct{}),
		ticker:  time.NewTicker(time.Second / time.Duration(stdTPS)),
		acu:     &acumulator{dt: make([]*Input, 64)},
	}
}

type Instance struct { // TODO should Instance have an ID?
	handler Server
	timeout time.Duration
	done    chan struct{}
	started bool
	ticker  *time.Ticker

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
			ins.handler.Update(ins)
			ins.disp.dispatch()
		case <-ins.done:
			return
		}
	}
}

func (ins *Instance) terminate() {
	ins.disp.killAll()
	ins.done <- struct{}{}
}

func (ins *Instance) GetData() []*Input {
	return ins.acu.consume()
}

func (ins *Instance) Broadcast(dt ...interface{}) {
	ins.disp.addDisp(ins.disp.currPlayers(), dt)
}

func (ins *Instance) Multicast(g *Group, dt ...interface{}) {
	ins.disp.addDisp(g, dt)
}

func (ins *Instance) Unicast(p *Player, dt ...interface{}) {
	ins.disp.addDisp(p, dt)
}
