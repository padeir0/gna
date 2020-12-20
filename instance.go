package gna

import (
	"time"
)

/*Instance is your game state, it's used to create an instance.
Each method runs concurrently with one another.
*/
type Instance interface {
	/*Update loop of your game state, you can get the batch of data from the Players
	with Instance.GetData(), and dispatch it with Instance.Dispatch*/
	Update()
	/*Disconn happens when a player disconnects*/
	Disconn(*Player)
	/*Auth happens when a player connects, to refuse the player connection
	simply close it: Player.Close(). To accept it, leave it be. The instance is
	the main instance of the server, if you do not manually set the instance with
	Player.SetInstance, the player is set to the main instance.
	*/
	Auth(*Player)
	/*NetAbs exposes the underlying networking abstraction,
	  this is for internal use only.*/
	NetAbs() *Net
	/*Terminate closes all connections inside the instance and stops the updates*/
	Terminate()
}

/*RunInstance starts the ticker and Disconnection Handler,
it's the only place where Instance.Update is called.
If RunInstance is called twice in a Instance it just returns.
*/
func RunInstance(ins Instance) {
	n := ins.NetAbs()
	if n.started {
		return
	}
	n.fillDefault()
	go dcHandler(n.dc, ins)
	n.started = true
	for {
		select {
		case <-n.ticker.C:
			ins.Update()
		case <-n.done:
			return
		}
	}
}

/*Net abstracts the Networking, it is meant to be embedded into your
server Instance, providing the Terminate and NetAbs methods.
*/
type Net struct {
	rTimeout time.Duration
	wTimeout time.Duration
	done     chan struct{}
	started  bool
	ticker   *time.Ticker

	Players *Group

	acu *acumulator
	dc  chan *Player
}

/*NetAbs exposes the underlying networking abstraction,
this is for internal use only.
*/
func (n *Net) NetAbs() *Net {
	return n
}

func (n *Net) fillDefault() {
	n.rTimeout = stdReadTimeout
	n.wTimeout = stdWriteTimeout
	n.done = make(chan struct{})
	n.ticker = time.NewTicker(time.Second / time.Duration(stdTPS))
	n.Players = &Group{pMap: make(map[uint64]*Player, 16)}
	n.acu = &acumulator{dt: make([]*Input, 64)}
	n.dc = make(chan *Player, 1)
}

/*Terminate closes the connection with all players and stops the updates*/
func (n *Net) Terminate() {
	n.Players.Close()
	n.done <- struct{}{}
}

/*GetData empties the Net acumulator, retrieving the Inputs*/
func (n *Net) GetData() []*Input {
	return n.acu.consume()
}

/*Dispatch sends the data to the corresponding dispatchers for each shipper,
the structs that implement ship() are: Players and Groups*/
func (*Net) Dispatch(s shipper, data interface{}) {
	s.ship(data)
}
