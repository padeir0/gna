package gna

import (
	"encoding/gob"
	"fmt"
	"log"
	"net"
	"sync"
	"time"
)

func newPlayer(id uint64, c net.Conn) *Player {
	return &Player{
		ID:          id,
		conn:        c,
		enc:         gob.NewEncoder(c),
		dec:         gob.NewDecoder(c),
		blep:        make(chan interface{}, 16),
		shouldStart: true,
	}
}

/*Player runs above a persistent TCP connection with gob encoding,
it acts as a receiver and owner of the connection.
*/
type Player struct {
	ID   uint64
	conn net.Conn
	enc  *gob.Encoder
	dec  *gob.Decoder
	wErr error
	rErr error

	blep chan interface{}

	ins         *Instance
	shouldStart bool // only used after auth, not concurrently
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

func (p *Player) ship(dt interface{}) {
	select {
	case p.blep <- dt:
	default:
		p.Close()
	}
}

func (p *Player) Send(dt interface{}) error {
	err := p.conn.SetWriteDeadline(time.Now().Add(2 * time.Second))
	if err != nil {
		return err
	}
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

/*Close terminates the player, closing the connection.*/
func (p *Player) Close() error {
	p.shouldStart = false // used in auth
	return p.conn.Close()
}

func (p *Player) disc() {
	p.ins.dc <- p
}

func (p *Player) start() {
	go p.ear()
	go p.mouth()
}

func (p *Player) mouth() {
	defer p.Close()
	for {
		dt := <-p.blep
		err := p.Send(dt)
		if err != nil {
			log.Println(err)
			return
		}
	}
}

func (p *Player) ear() {
	defer p.disc()
	defer p.Close()
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
		p.Close()
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
func (g *Group) ship(data interface{}) {
	g.mu.Lock()
	for _, p := range g.pMap {
		p.ship(data)
	}
	g.mu.Unlock()
}

/*Returns the number of players in the group*/
func (g *Group) Len() int {
	g.mu.Lock()
	out := len(g.pMap)
	g.mu.Unlock()
	return out
}
