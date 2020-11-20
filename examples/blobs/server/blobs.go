package main

import (
	"github.com/kazhmir/mgs"
	"github.com/kazhmir/mgs/examples/blobs/shared"
	"log"
)

var password = []byte("MyPassWord")
var state = NewGameState(500)
var evl = &EventList{list: make([]mgs.Encoder, 128)}
var server *mgs.Server

func main() {
	server = &mgs.Server{
		Addr:          "localhost:8888",
		Logic:         GameLogic,
		Unmarshaler:   Protocol,
		Validate:      Validate,
		Disconnection: DeSpawn,
		Verbose:       true,
	}
	log.Fatal(server.Start())
}

func GameLogic(dt []*mgs.Input) map[mgs.Sender][]mgs.Encoder {
	out := evl.Consume()
	out = append(out, make([]mgs.Encoder, server.AllTalkers().Len())...)
	var n int
	for i := range dt {
		v, _ := dt[i].Data.(Keys)
		b := state.blobs[dt[i].T.ID]
		for j := range v {
			switch v[j] {
			case 'w':
				b.Move(false)
			case 's':
				b.Move(true)
			case 'a':
				b.Rotate(false)
			case 'd':
				b.Rotate(true)
			}
		}
		if n >= len(out) {
			out = append(out, make([]mgs.Encoder, 10)...)
		}
		out[n] = b
		n++
	}
	return map[mgs.Sender][]mgs.Encoder{server.AllTalkers(): out}
}

func Protocol(p *mgs.Packet) interface{} {
	out := make([]byte, 4)
	var n int
	for i := range p.Data {
		switch p.Data[i] {
		case 'w', 'a', 's', 'd':
			if n >= 5 {
				break
			}
			out[n] = p.Data[i]
			n++
		}
	}
	return Keys(out[:n])
}

func Validate(id uint64, b *mgs.Packet) (mgs.Encoder, bool) {
	if len(b.Data) != len(password) {
		return nil, false
	}
	for i := range b.Data {
		if b.Data[i] != password[i] {
			return nil, false
		}
	}
	evl.Add(&shared.Event{ID: id, T: shared.EBorn})
	return state.NewBlob(id), true
}

func DeSpawn(id uint64) {
	state.RmBlob(id)
	evl.Add(&shared.Event{ID: id, T: shared.EDied})
}
