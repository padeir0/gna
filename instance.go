package gna

import (
	"log"
	"time"
)

type World interface {
	Update(*Instance)
	Disconn(*Instance, *Player)
	Auth(*Instance, *Player)
}

func NewInstance(sr World) *Instance {
	return &Instance{
		world:    sr,
		rTimeout: stdReadTimeout,
		wTimeout: stdWriteTimeout,
		done:     make(chan struct{}),
		ticker:   time.NewTicker(time.Second / time.Duration(stdTPS)),
		players:  &Group{pMap: make(map[uint64]*Player, 16)},
		acu:      &acumulator{dt: make([]*Input, 64)},
		disp:     make(chan *packet, 1),
		dc:       make(chan *Player, 1),
	}
}

type Instance struct { // TODO should Instance have an ID?
	world    World
	rTimeout time.Duration
	wTimeout time.Duration
	done     chan struct{}
	started  bool
	ticker   *time.Ticker

	players *Group

	acu  *acumulator
	disp chan *packet
	dc   chan *Player
}

func (ins *Instance) Start() {
	if ins.started {
		return
	}
	for i := 0; i < 5; i++ {
		go dispatcher(ins.disp)
	}
	go dcHandler(ins.dc, ins)
	ins.started = true
	for {
		select {
		case <-ins.ticker.C:
			ins.world.Update(ins)
		case <-ins.done:
			return
		}
	}
}

func (ins *Instance) terminate() {
	log.Printf("\n%v\n", ins.players)
	ins.players.Close()
	ins.done <- struct{}{}
}

func (ins *Instance) GetData() []*Input {
	return ins.acu.consume()
}

func (ins *Instance) Broadcast(dt interface{}) {
	ins.disp <- &packet{ins.players, dt}
}

func (ins *Instance) Multicast(g *Group, dt interface{}) {
	ins.disp <- &packet{g, dt}
}

func (ins *Instance) Unicast(p *Player, dt interface{}) {
	ins.disp <- &packet{p, dt}
}
