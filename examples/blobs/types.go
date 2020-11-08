package main

import (
	"encoding/binary"
	"github.com/kazhmir/mgs"
	"math"
	"math/rand"
	"sync"
)

func NewGameState(size int) *GameState {
	return &GameState{
		make(map[uint32]*Blob, 64),
		size,
		sync.Mutex{},
	}
}

type GameState struct {
	blobs map[uint32]*Blob
	size  int
	mu    sync.Mutex
}

func (gm *GameState) NewBlob(id uint32) *Point {
	b := &Blob{}
	gm.mu.Lock()
	gm.blobs[id] = b
	gm.mu.Unlock()
	return b.Spawn(gm.size)
}

func (gm *GameState) RmBlob(id uint32) {
	gm.mu.Lock()
	delete(gm.blobs, id)
	gm.mu.Unlock()
}

type Blob struct {
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

func (b *Blob) MarshalBinary() ([]byte, error) {
	buff := make([]byte, 12)
	binary.BigEndian.PutUint32(buff, math.Float32bits(float32(b.p.x)))
	binary.BigEndian.PutUint32(buff[4:], math.Float32bits(float32(b.p.y)))
	binary.BigEndian.PutUint32(buff[8:], math.Float32bits(float32(b.rot)))
	return buff, nil
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

func (p *Point) MarshalBinary() ([]byte, error) {
	buff := make([]byte, 8)
	binary.BigEndian.PutUint32(buff, math.Float32bits(float32(p.x)))
	binary.BigEndian.PutUint32(buff[4:], math.Float32bits(float32(p.y)))
	return buff, nil
}

func (p *Point) Randomize(max int) {
	p.x = float64(rand.Intn(max))
	p.y = float64(rand.Intn(max))
}

type Keys string

func (k Keys) MarshalBinary() ([]byte, error) {
	return []byte(k), nil
}

type Data []byte

func (dt Data) MarshalBinary() ([]byte, error) {
	return dt, nil
}

type inputByTime []*mgs.Input

func (inp *inputByTime) Len() int {
	return len(*inp)
}

func (inp *inputByTime) Less(i, j int) bool {
	return (*inp)[i].Time < (*inp)[j].Time
}

func (inp *inputByTime) Swap(i, j int) {
	c := (*inp)[i]
	(*inp)[i] = (*inp)[j]
	(*inp)[j] = c
}
