package gna

import (
	"encoding/gob"
	"errors"
	"fmt"
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

/*Player runs above a persistent TCP connection with gob encoding,
it abstracts the protocol and network logic.
*/
type Player struct {
	/*Talker ID*/
	ID   uint64
	conn net.Conn
	enc  *gob.Encoder
	dec  *gob.Decoder
	wErr error
	rErr error

	ins *Instance

	mouthDt chan []interface{}

	started bool

	// assure safe termination
	dead bool
	mu   sync.Mutex
}

/*Removes the player from the previous instance, if any,
and sends him to another.*/
func (p *Player) SetInstance(ins *Instance) {
	if p.ins != nil {
		p.ins.players.Rm(p.ID)
	}
	p.ins = ins
	p.ins.players.Add(p)
}

func (p *Player) Send(dt interface{}) error {
	return p.enc.Encode(&dt)
}

func (p *Player) Recv() (interface{}, error) {
	var dt interface{}
	err := p.dec.Decode(&dt)
	return dt, err
}

func (p *Player) Error() error {
	if p.wErr != nil {
		if p.rErr != nil {
			return fmt.Errorf("%w, along with: %v", p.wErr, p.rErr)
		}
		return p.wErr
	}
	if p.rErr != nil {
		return p.rErr
	}
	return nil
}

/*send sends the data to the mouth of the player*/
func (p *Player) send(dt []interface{}) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.dead {
		p.mouthDt <- dt
	}
}

/*Terminate terminates the player, executing Disconnect,
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
		p.ins.world.Disconn(p.ins, p)
		p.conn.Close()
		p.ins.players.Rm(p.ID)
		close(p.mouthDt)
	}
}

/*Start starts the player, creating 2 new goroutines.
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
		if dt == nil {
			return
		}
		for i := range dt {
			err := p.Send(dt[i])
			if err != nil {
				p.wErr = fmt.Errorf("%w while encoding: %v", err, dt[i])
				if errors.Is(err, syscall.EPIPE) { // should return without logging
					return
				}
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
			p.rErr = fmt.Errorf("failed to set deadline: %w", err)
		}
		dt, err := p.Recv()
		if err != nil {
			p.rErr = fmt.Errorf("recv: %w", err)
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

func NewGroup(ps ...*Player) *Group {
	pMap := make(map[uint64]*Player, len(ps))
	for i := range ps {
		pMap[ps[i].ID] = ps[i]
	}
	return &Group{pMap: pMap}
}

/*Group is a collection of players that is safe for concurrent use,
This can be used to "multicast" a single piece o data to a set of players.
*/
type Group struct {
	pMap map[uint64]*Player
	mu   sync.Mutex
}

func (g *Group) Terminate() {
	g.mu.Lock()
	for _, p := range g.pMap {
		p.mu.Lock()
		p.dead = true
		p.conn.Close()
		close(p.mouthDt)
		p.mu.Unlock()
	}
	g.pMap = nil
	g.mu.Unlock()
}

/*Add a player to the Group*/
func (g *Group) Add(t *Player) {
	g.mu.Lock()
	g.pMap[t.ID] = t
	g.mu.Unlock()
}

/*Rm removes a player from the Group*/
func (g *Group) Rm(id uint64) {
	g.mu.Lock()
	delete(g.pMap, id)
	g.mu.Unlock()
}

/*Send sends the sig channel and data to each Talker in the group*/
func (g *Group) send(data []interface{}) {
	for _, p := range g.pMap {
		p.send(data)
	}
}

/*Returns the number of players in the group*/
func (g *Group) Len() int {
	g.mu.Lock()
	out := len(g.pMap)
	g.mu.Unlock()
	return out
}
