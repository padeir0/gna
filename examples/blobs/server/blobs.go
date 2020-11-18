package main

import (
	"github.com/kazhmir/mgs"
	"log"
)

var password = []byte("MyPassWord")
var state = NewGameState(500)
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
	}
	return map[mgs.Sender][]mgs.Encoder{server.AllTalkers(): state.Blobs()}
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
	return state.NewBlob(id), true
}

func DeSpawn(id uint64) {
	state.RmBlob(id)
}
