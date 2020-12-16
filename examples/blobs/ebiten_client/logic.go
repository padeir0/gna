package main

import (
	"fmt"
	"github.com/hajimehoshi/ebiten"
	//	"github.com/hajimehoshi/ebiten/ebitenutil"
	"github.com/kazhmir/gna"
	"github.com/kazhmir/gna/examples/blobs/shared"
	"log"
	//	"math"
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
	input := ""
	if ebiten.IsKeyPressed(ebiten.KeyS) {
		input += "s"
	}
	if ebiten.IsKeyPressed(ebiten.KeyW) {
		input += "w"
	}
	if ebiten.IsKeyPressed(ebiten.KeyD) {
		input += "d"
	}
	if ebiten.IsKeyPressed(ebiten.KeyA) {
		input += "a"
	}
	if ebiten.IsKeyPressed(ebiten.KeySpace) {
		fmt.Println(len(g.blobs))
	}
	if input != "" {
		g.conn.Send(input)
		if err := g.conn.Error(); err != nil {
			log.Fatal(err)
		}
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

func (g *Game) ServerUpdate() {
	dt := g.conn.RecvBatch()
	if len(dt) == 0 {
		return
	}
	if err := g.conn.Error(); err != nil {
		log.Fatal(err)
	}
	for _, data := range dt {
		switch v := data.(type) {
		case shared.Blob:
			if _, ok := g.blobs[v.ID]; ok {
				g.blobs[v.ID].Blob = v
				continue
			}
			g.blobs[v.ID] = &Blob{d: D, img: ball, Blob: v}
		case []*shared.Blob:
			for i := range v {
				if _, ok := g.blobs[v[i].ID]; ok {
					g.blobs[v[i].ID].Blob = *v[i]
					continue
				}
				g.blobs[v[i].ID] = &Blob{d: D, img: ball, Blob: *v[i]}
			}
		case shared.Event:
			switch v.T {
			case shared.EDied:
				g.RmBlob(v.ID)
			case shared.EBorn:
				b := &Blob{
					d:    D,
					img:  ball,
					Blob: shared.Blob{ID: v.ID},
				}
				g.AddBlob(b)
			}
		default:
			fmt.Printf("\n%v\n", data)
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
	//	sin, cos := math.Sincos(ent.Blob.Rot)
	//	s := fmt.Sprintf("x: %v, y: %v, rot: %v\nsin: %v, cos: %v", ent.Blob.P.X, ent.Blob.P.Y, ent.Blob.Rot, sin, cos)
	//	ebitenutil.DebugPrint(screen, s)
}
