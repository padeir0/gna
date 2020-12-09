package mgs

import (
	"encoding/gob"
	"errors"
	"log"
	"net"
	"sync"
	"syscall"
	"time"
)

func newPlayer(id uint64, c net.Conn) *Player {
	return &Player{
		ID:      id,
		conn:    c,
		enc:     gob.NewEncoder(c),
		dec:     gob.NewDecoder(c),
		mouthDt: make(chan []interface{}),
	}
}

/*Player abstracts the protocol and network logic.
 */
type Player struct {
	/*Talker ID*/
	ID   uint64
	conn net.Conn
	enc  *gob.Encoder
	dec  *gob.Decoder

	ins *Instance

	mouthDt chan []interface{}

	started bool

	// assure safe termination
	dead bool
	mu   sync.Mutex
}

func (p *Player) SetInstance(ins *Instance) {
	if p.ins != nil {
		p.ins.disp.rmPlayer(p.ID)
	}
	p.ins = ins
	p.ins.disp.addPlayer(p)
}

func (p *Player) Send(dt interface{}) error {
	return p.enc.Encode(&dt)
}

func (p *Player) Recv() (interface{}, error) {
	var dt interface{}
	err := p.dec.Decode(&dt)
	return dt, err
}

/*Rectify returns true if the talker exists in the map*/
func (p *Player) rectify(mp *map[uint64]*Player) bool {
	if _, ok := (*mp)[p.ID]; !ok {
		return false
	}
	return true
}

/*send sends the signal channel and data to the mouth of the talker*/
func (p *Player) send(dt []interface{}) {
	p.mouthDt <- dt
}

/*Terminate terminates the talker, executing Disconnect,
closing the connection and the channels.*/
func (p *Player) Terminate() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.started {
		p.dead = true
		p.conn.Close()
		return
	}
	if !p.dead {
		p.dead = true
		p.ins.handler.Disconn(p)
		p.conn.Close()
		p.ins.disp.rmPlayer(p.ID)
		close(p.mouthDt)
	}
}

/*Start starts the talker, creating 2 new goroutines.
The ear goroutine listens for packets and sents these packets through the ins.Out channel.
The mouth goroutine waits for responses from the dispatcher and writes them to the connection.
*/
func (p *Player) start() {
	p.started = true
	go p.mouth()
	go p.ear()
}

func (p *Player) mouth() {
	dt := []interface{}{}
	defer p.Terminate()
	for {
		dt = <-p.mouthDt
		for i := range dt {
			err := p.Send(dt[i])
			if errors.Is(err, syscall.EPIPE) { // ?
				log.Println("Cancelling packets to: ", p.ID, ".", err)
				return
			}
			if err != nil {
				log.Println(err)
				break
			}

		}
	}
}

func (p *Player) ear() {
	defer p.Terminate()
	for {
		err := p.conn.SetReadDeadline(time.Now().Add(p.ins.timeout))
		if err != nil {
			log.Println(err)
		}
		dt, err := p.Recv()
		if err != nil {
			log.Println(err)
			/*if opErr, ok := err.(*net.OpError); ok && opErr.Timeout() {
				return
			}*/
			return
		}
		if dt != nil { // ?
			p.ins.acu.add(&Input{p, dt})
		}
	}
}

/*Group is a collection of talkers that is safe for concurrent use,
This can be used to "multicast" a single piece o data to a set of talkers.
*/
type Group struct {
	tMap map[uint64]*Player
	mu   sync.Mutex
}

/*Add a talker to the Group*/
func (g *Group) Add(t *Player) {
	g.mu.Lock()
	g.tMap[t.ID] = t
	g.mu.Unlock()
}

/*Rm removes a talker from the Group*/
func (g *Group) Rm(id uint64) {
	g.mu.Lock()
	delete(g.tMap, id)
	g.mu.Unlock()
}

/*Rectify removes talkers from the Group that are not in the given map*/
func (g *Group) rectify(mp *map[uint64]*Player) bool {
	for id := range *mp {
		if _, ok := g.tMap[id]; !ok {
			g.Rm(id)
		}
	}
	if len(g.tMap) == 0 {
		return false
	}
	return true
}

/*Send sends the sig channel and data to each Talker in the group*/
func (g *Group) send(data []interface{}) {
	for _, t := range g.tMap {
		t.mouthDt <- data
	}
}

func (g *Group) Len() int {
	g.mu.Lock()
	out := len(g.tMap)
	g.mu.Unlock()
	return out
}
