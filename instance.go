package gna

import (
	"time"
)

/*World is your game state, it's used to create an instance.
Each method runs concurrently with one another.
*/
type World interface {
	/*Update loop of your game state, you can get the batch of data from the Players
	with Instance.GetData(), and dispatch it with Instance.Dispatch*/
	Update(*Instance)
	/*Disconn happens when a player disconnects*/
	Disconn(*Instance, *Player)
	/*Auth happens when a player connects, to refuse the player connection
	simply close it: Player.Close(). To accept it, leave it be. The instance is the
	main instance of the server, if you do not manually set the instance with Player.SetInstance
	the player is set to the main instance.
	*/
	Auth(*Instance, *Player)
}

/*NewInstance wraps the World with a instance, you'll still need to start it
with Instance.Start*/
func NewInstance(sr World) *Instance {
	return &Instance{
		world:    sr,
		rTimeout: stdReadTimeout,
		wTimeout: stdWriteTimeout,
		done:     make(chan struct{}),
		ticker:   time.NewTicker(time.Second / time.Duration(stdTPS)),
		Players:  &Group{pMap: make(map[uint64]*Player, 16)},
		acu:      &acumulator{dt: make([]*Input, 64)},
		disp:     make(chan *packet, 1),
		dc:       make(chan *Player, 1),
	}
}

/*Instance wraps a game state and carry necessary methods to
get and dispatch data.
*/
type Instance struct {
	world    World
	rTimeout time.Duration
	wTimeout time.Duration
	done     chan struct{}
	started  bool
	ticker   *time.Ticker

	Players *Group

	acu  *acumulator
	disp chan *packet
	dc   chan *Player
}

/*Start starts the ticker and handlers, it's the only place where World.Update is called*/
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

/*Terminate closes the connection with all players and stops the updates*/
func (ins *Instance) Terminate() {
	ins.Players.Close()
	ins.done <- struct{}{}
}

/*GetData empties the Instance acumulator, retrieving the Inputs*/
func (ins *Instance) GetData() []*Input {
	return ins.acu.consume()
}

/*Dispatch sends the data to the corresponding dispatchers for each shipper,
the structs that implement ship() are: Players and Groups*/
func (ins *Instance) Dispatch(s shipper, data interface{}) {
	ins.disp <- &packet{s, data}
}
