package main

import (
	"fmt"
	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/ebitenutil"
	"github.com/kazhmir/gna"
	"github.com/kazhmir/gna/examples/blobs/shared"
	"math"
	"sync"
)

type Game struct {
	playerID uint64
	blobs    map[uint64]*Blob
	mu       sync.Mutex
	conn     *gna.Client
}

func (g *Game) Update() error {
	g.ServerUpdate()
	if ebiten.IsKeyPressed(ebiten.KeyS) {
		g.Player().Move(true)
	}
	if ebiten.IsKeyPressed(ebiten.KeyW) {
		g.Player().Move(false)
	}
	if ebiten.IsKeyPressed(ebiten.KeyD) {
		g.Player().Rotate(true)
	}
	if ebiten.IsKeyPressed(ebiten.KeyA) {
		g.Player().Rotate(false)
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	for i := range g.blobs {
		g.blobs[i].draw(screen)
	}
}

func (g *Game) Layout(outWid, outHei int) (int, int) {
	return scrWid, scrHei
}

func (g *Game) AddBlob(b *Blob) {
	g.mu.Lock()
	g.blobs[b.ID] = b
	g.mu.Unlock()
}

func (g *Game) RmBlob(id uint64) {
	g.mu.Lock()
	delete(g.blobs, id)
	g.mu.Unlock()
}

func (g *Game) Player() *Blob {
	return g.blobs[g.playerID]
}

func (g *Game) ServerUpdate() error {
	g.mu.Lock()
	defer g.mu.Unlock()
	for _, data := range g.conn.RecvAll() {
		switch v := data.(type) {
		case shared.Blob:
			g.blobs[v.ID] = v
		case shared.Event:
			switch v.T {
			case shared.EDied:
				g.RmBlob(v.ID)
			case shared.EBorn:
				b := &Blob{
					d:    D,
					img:  ball,
					blob: {ID: v.ID},
				}
				g.AddBlob()
			}
		}
	}
}

type Blob struct {
	d   float64
	img *ebiten.Image
	shared.Blob
}

func (ent *Blob) draw(screen *ebiten.Image) {
	newOp := &ebiten.DrawImageOptions{}
	offsetx := ent.d / 2
	offsety := ent.d / 2
	newOp.GeoM.Translate(-offsetx, -offsety)
	newOp.GeoM.Rotate(ent.Blob.Rot)
	newOp.GeoM.Translate(ent.Blob.P.X+offsetx, ent.Blob.P.Y+offsety)

	screen.DrawImage(ent.img, newOp)
	sin, cos := math.Sincos(ent.Blob.Rot)
	s := fmt.Sprintf("x: %v, y: %v, rot: %v\nsin: %v, cos: %v", ent.Blob.P.X, ent.Blob.P.Y, ent.Blob.Rot, sin, cos)
	ebitenutil.DebugPrint(screen, s)
}
