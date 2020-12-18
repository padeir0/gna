package gna

import (
	"encoding/gob"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"
)

func newPlayer(id uint64, c net.Conn) *Player {
	return &Player{
		ID:          id,
		conn:        c,
		rTimeout:    stdReadTimeout,
		wTimeout:    stdWriteTimeout,
		enc:         gob.NewEncoder(c),
		dec:         gob.NewDecoder(c),
		cDisp:       make(chan interface{}, 32),
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
	wErr error // encode/write error
	rErr error // decode/read error

	cDisp chan interface{}

	dc          chan *Player // chan to the disconnection handler
	grp         *Group       // instance group
	acu         *acumulator  // player acumulator, shared with other players in the instance
	rTimeout    time.Duration
	wTimeout    time.Duration
	shouldStart bool // only used after auth, not concurrently
}

/*SetInstance removes the player from the previous instance, if any,
and sends him to another.*/
func (p *Player) SetInstance(ins *Instance) {
	if p.grp != nil {
		p.grp.Rm(p.ID)
	}
	p.grp = ins.Players
	p.grp.Add(p)
	p.acu = ins.acu
	p.rTimeout = ins.rTimeout
	p.wTimeout = ins.wTimeout
	p.dc = ins.dc
}

func (p *Player) ship(dt interface{}) {
	select {
	case p.cDisp <- dt:
	default:
		p.wErr = errors.New("full tcp buffer")
		p.Close() // this means the TCP buffer of the receiver is full
	}
}

/*Send sets the deadline and encodes the data, it may halt, but it returns the error
in case of failure, guaranteeing knowledge if the user has the data.
This differs from Player.ship() in which it's only known after the connection is closed.*/
func (p *Player) Send(dt interface{}) error {
	err := p.conn.SetWriteDeadline(time.Now().Add(15 * time.Second))
	if err != nil {
		return err
	}
	return p.enc.Encode(&dt)
}

/*Recv sets the deadline and decodes data from the connection, you cannot use this safely
after Auth, as the goroutine responsible for receiving data is started.*/
func (p *Player) Recv() (interface{}, error) {
	err := p.conn.SetReadDeadline(time.Now().Add(p.rTimeout))
	if err != nil {
		err = fmt.Errorf("failed to set deadline: %w", err)
		return nil, err
	}
	var dt interface{}
	err = p.dec.Decode(&dt)
	return dt, err
}

/*Error returns the error that caused the Player to disconnect.*/
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
	p.dc <- p
}

func (p *Player) start() {
	go p.ear()
	go p.mouth()
}

func (p *Player) mouth() {
	defer p.Close()
	for {
		dt := <-p.cDisp
		err := p.Send(dt)
		if err != nil {
			if p.wErr == nil {
				p.wErr = err
			}
			return
		}
	}
}

func (p *Player) ear() {
	defer p.disc()
	defer p.Close()
	for {
		dt, err := p.Recv()
		if err != nil {
			p.rErr = fmt.Errorf("recv: %w", err)
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

/*NewGroup creates a group containing the specified players*/
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

/*Close closes all players inside the Group and frees the map for garbage
collection.*/
func (g *Group) Close() {
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

/*ship sends the sig channel and data to each Talker in the group*/
//TODO: This method makes each Player.ear do duplicated work, it should be optimized to encode only once
func (g *Group) ship(data interface{}) {
	g.mu.Lock()
	for _, p := range g.pMap {
		p.ship(data)
	}
	g.mu.Unlock()
}

/*Len returns the number of players in the group*/
func (g *Group) Len() int {
	g.mu.Lock()
	out := len(g.pMap)
	g.mu.Unlock()
	return out
}
