package shared

import (
	"encoding/binary"
	"math"
	"math/rand"
)

const (
	TPoint = byte(iota + 1)
	TBlob
	TEvent
)

const (
	EDied = byte(iota)
	EBorn
)

type Blob struct {
	id  uint64
	p   Point
	rot float64
}

func (b *Blob) Rotate(back bool) {
	if back {
		b.rot -= math.Pi / 12
	} else {
		b.rot += math.Pi / 12
	}
}

func (b *Blob) Move(back bool) {
	if back {
		b.p.Translate(b.rot, -5)
	} else {
		b.p.Translate(b.rot, 5)
	}
}

func (b *Blob) Spawn(max int) *Point {
	b.p.Randomize(max)
	return &b.p
}

func (b *Blob) Size() int {
	return 20
}

func (b *Blob) Type() byte {
	return TBlob
}

func (b *Blob) Encode(buff []byte) error {
	binary.BigEndian.PutUint64(buff, b.id)
	binary.BigEndian.PutUint32(buff[8:], math.Float32bits(float32(b.p.x)))
	binary.BigEndian.PutUint32(buff[12:], math.Float32bits(float32(b.p.y)))
	binary.BigEndian.PutUint32(buff[16:], math.Float32bits(float32(b.rot)))
	return nil
}

type Point struct {
	x, y float64
}

func (p1 *Point) Distance(p2 Point) float64 {
	b := p1.x - p2.x
	c := p1.y - p2.y
	return math.Sqrt(b*b + c*c)
}

func (p *Point) Translate(angle float64, amount float64) {
	sin, cos := math.Sincos(angle)
	p.x += amount * sin
	p.y -= amount * cos
}

func (p *Point) Size() int {
	return 8
}

func (p *Point) Type() byte {
	return TPoint
}

func (p *Point) Encode(buff []byte) error {
	binary.BigEndian.PutUint32(buff, math.Float32bits(float32(p.x)))
	binary.BigEndian.PutUint32(buff[4:], math.Float32bits(float32(p.y)))
	return nil
}

func (p *Point) Randomize(max int) {
	p.x = float64(rand.Intn(max))
	p.y = float64(rand.Intn(max))
}

type Event struct {
	ID uint64
	T  byte
}

func (e *Event) Size() int {
	return 9
}

func (e *Event) Type() byte {
	return TEvent
}

func (e *Event) Encode(buff []byte) error {
	binary.BigEndian.PutUint64(buff, e.ID)
	buff[8] = e.T
	return nil
}
