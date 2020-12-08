package main

import (
	"fmt"
	"github.com/kazhmir/mgs"
)

var server mgs.Server

func main() {
}

func GameLogic(dt []*mgs.Input) map[mgs.Sender][]mgs.Encoder {
	all := server.AllTalkers()
	out := map[mgs.Sender][]mgs.Encoder{
		all: make([]mgs.Encoder, len(dt)),
	}
	for i := range dt {
		v := dt[i].Data.(Data)
		out[all][i] = v
	}
	return out
}

func Protocol(p *mgs.Packet) interface{} {
	return Data(p.Data)
}

func Validate(id uint64, p *mgs.Packet) (mgs.Encoder, bool) {
	return Data(p.Data), true
}

func Disconnection(id uint64) {
	fmt.Println(id, "Disconnected")
}

type Data []byte

func (dt Data) Size() int {
	return len(dt)
}

func (dt Data) Type() byte {
	return 2
}

func (dt Data) Encode(buff []byte) error {
	for i := range dt {
		buff[i] = dt[i]
	}
	return nil
}
