package main

import (
	"github.com/kazhmir/mgs"
	"github.com/kazhmir/mgs/examples/blobs/shared"
	"sync"
)

func NewGameState(size int) *GameState {
	return &GameState{
		make(map[uint64]*shared.Blob, 64),
		size,
		sync.Mutex{},
	}
}

type GameState struct {
	blobs map[uint64]*shared.Blob
	size  int
	mu    sync.Mutex
}

func (gm *GameState) NewBlob(id uint64) *shared.Point {
	b := &shared.Blob{}
	gm.mu.Lock()
	gm.blobs[id] = b
	gm.mu.Unlock()
	return b.Spawn(gm.size)
}

func (gm *GameState) RmBlob(id uint64) {
	gm.mu.Lock()
	delete(gm.blobs, id)
	gm.mu.Unlock()
}

func (gm *GameState) AllBlobs() []mgs.Encoder {
	gm.mu.Lock()
	out := make([]mgs.Encoder, len(gm.blobs))
	i := 0
	for _, b := range gm.blobs {
		out[i] = b
		i++
	}
	gm.mu.Unlock()
	return out
}

type EventList struct {
	list []mgs.Encoder
	i    int
	mu   sync.Mutex
}

func (evl *EventList) Add(e mgs.Encoder) {
	evl.mu.Lock()
	if evl.i >= len(evl.list) {
		evl.list = append(evl.list, make([]mgs.Encoder, 64)...)
	}
	evl.list[evl.i] = e
	evl.i++
	evl.mu.Unlock()
}

func (evl *EventList) Consume() []mgs.Encoder {
	out := make([]mgs.Encoder, len(evl.list))
	copy(out, evl.list)
	evl.i = 0
	return out
}

func (evl *EventList) Len() int {
	return len(evl.list)
}

type Keys string

func (k Keys) MarshalBinary() ([]byte, error) {
	return []byte(k), nil
}

type Data []byte

func (dt Data) MarshalBinary() ([]byte, error) {
	return dt, nil
}
