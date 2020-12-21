package gna

import (
	"fmt"
	"time"
)

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

	acu *playerBucket
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
	n.acu = &playerBucket{dt: make([]*Input, 64)}
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
