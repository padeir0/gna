package main

import (
	"encoding"
	"github.com/kazhmir/mgs"
	"log"
	"sort"
)

var password = []byte("MyPassWord")
var state = NewGameState(500)

func main() {
	server := mgs.Server{
		Addr:          "localhost:8888",
		Logic:         GameLogic,
		Unmarshaler:   Protocol,
		Validate:      Validate,
		Disconnection: DeSpawn,
		Verbose:       true,
	}
	log.Fatal(server.Start())
}

func GameLogic(dt []*mgs.Input) map[uint32][]encoding.BinaryMarshaler {
	inp := inputByTime(dt)
	sort.Sort(&inp)
	out := make(map[uint32][]encoding.BinaryMarshaler, len(state.blobs))
	for i := range inp {
		v, _ := inp[i].Data.(Keys)
		b := state.blobs[inp[i].T.Id]
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
	for k1 := range state.blobs {
		out[k1] = make([]encoding.BinaryMarshaler, len(state.blobs))
		i := 0
		for _, b := range state.blobs {
			out[k1][i] = b
			i++
		}
		out[k1] = out[k1][:i]
	}
	return out
}

func Protocol(b []byte) encoding.BinaryMarshaler {
	out := make([]byte, 4)
	var n int
	for i := range b {
		switch b[i] {
		case 'w', 'a', 's', 'd':
			if n >= 5 {
				break
			}
			out[n] = b[i]
			n++
		}
	}
	return Keys(out[:n])
}

func Validate(id uint32, b []byte) (encoding.BinaryMarshaler, bool) {
	if len(b) != len(password) {
		return nil, false
	}
	for i := range b {
		if b[i] != password[i] {
			return nil, false
		}
	}
	return state.NewBlob(id), true
}

func DeSpawn(id uint32) {
	state.RmBlob(id)
}
