package main

import (
	"fmt"
	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/ebitenutil"
	"math"
	"math/rand"
)

type Game struct {
	player *entity
	others []*entity
}

func (g *Game) Update() error {
	if ebiten.IsKeyPressed(ebiten.KeyS) {
		g.player.moveBackward()
	}
	if ebiten.IsKeyPressed(ebiten.KeyW) {
		g.player.moveFoward()
	}
	if ebiten.IsKeyPressed(ebiten.KeyD) {
		g.player.rotateRight()
	}
	if ebiten.IsKeyPressed(ebiten.KeyA) {
		g.player.rotateLeft()
	}
	if ebiten.IsKeyPressed(ebiten.KeySpace) {
		g.player.fire()
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	if g.player.alive {
		g.player.draw(screen)
	}
	for i := range g.others {
		if g.others[i].alive {
			g.others[i].draw(screen)
		}
	}
}

func (g *Game) Layout(outWid, outHei int) (int, int) {
	return scrWid, scrHei
}

func newEntity(x, y float64, evil bool) *entity {
	out := &entity{
		x: float64(rand.Intn(scrWid - 50)),
		y: float64(rand.Intn(scrHei - 50)),
	}
	if evil {
		out.img = evilBall
		out.w, out.h = evilBall.Size()
	} else {
		out.img = ball
		out.w, out.h = ball.Size()
	}
	return out
}

type entity struct {
	x, y, rot          float64
	oldX, oldY, oldRot float64
	w, h               int
	health             int
	img                *ebiten.Image
	alive              bool
}

func (ent *entity) spawn(x, y float64) {
	ent.alive = true
	ent.x = x
	ent.y = y
}

func (ent *entity) deSpawn() {
	ent.alive = false
}

func (ent *entity) draw(screen *ebiten.Image) {
	newOp := &ebiten.DrawImageOptions{}
	//x, y, rot := ent.getMovement()
	offsetx := float64(ent.w) / 2
	offsety := float64(ent.h) / 2
	newOp.GeoM.Translate(-offsetx, -offsety)
	newOp.GeoM.Rotate(ent.rot)
	newOp.GeoM.Translate(ent.x+offsetx, ent.y+offsety)

	screen.DrawImage(ent.img, newOp)
	sin, cos := math.Sincos(ent.rot)
	s := fmt.Sprintf("x: %v, y: %v, rot: %v\nsin: %v, cos: %v", ent.x, ent.y, ent.rot, sin, cos)
	ebitenutil.DebugPrint(screen, s)
}

func (ent *entity) getMovement() (x float64, y float64, rot float64) {
	x = ent.x - ent.oldX
	y = ent.y - ent.oldY
	rot = ent.rot - ent.oldRot
	ent.oldX = ent.x
	ent.oldY = ent.y
	ent.oldRot = ent.rot
	return x, y, rot
}

func (ent *entity) moveFoward() {
	sin, cos := math.Sincos(ent.rot)
	ent.x += sin * 5
	ent.y -= cos * 5
}

func (ent *entity) moveBackward() {
	sin, cos := math.Sincos(ent.rot)
	ent.x -= sin * 5
	ent.y += cos * 5
}

func (ent *entity) rotateLeft() {
	ent.rot -= math.Pi / 30
}

func (ent *entity) rotateRight() {
	ent.rot += math.Pi / 30
}

func (ent *entity) fire() {
}
