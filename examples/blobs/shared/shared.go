package shared

import (
	"math"
	"math/rand"
)

const (
	EDied = byte(iota)
	EBorn
)

type Event struct {
	ID uint64
	T  byte
}

type Blob struct {
	ID  uint64
	P   Point
	Rot float64
}

func (b *Blob) Rotate(right bool) {
	if right {
		b.Rot += math.Pi / 24
	} else {
		b.Rot -= math.Pi / 24
	}
}

func (b *Blob) Move(back bool) {
	if back {
		b.P.Translate(b.Rot, -5)
	} else {
		b.P.Translate(b.Rot, 5)
	}
}

func (b *Blob) Spawn(max int) *Blob {
	b.P.Randomize(max)
	return b
}

type Point struct {
	X, Y float64
}

func (p1 *Point) Distance(p2 Point) float64 {
	b := p1.X - p2.X
	c := p1.Y - p2.Y
	return math.Sqrt(b*b + c*c)
}

func (p *Point) Translate(angle float64, amount float64) {
	sin, cos := math.Sincos(angle)
	p.X += amount * sin
	p.Y -= amount * cos
}

func (p *Point) Randomize(max int) {
	p.X = float64(rand.Intn(max))
	p.Y = float64(rand.Intn(max))
}
