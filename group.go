package gna

import "sync"

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
