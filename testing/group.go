package main

import (
	"fmt"
	"github.com/kazhmir/mgs"
	"time"
)

var server mgs.Server

func main() {
	server = mgs.Server{
		Addr:          "localhost:8888",
		Timeout:       time.Second * 10,
		TickInterval:  time.Millisecond * 100,
		Logic:         GameLogic,
		Unmarshaler:   Protocol,
		Validate:      Validate,
		Disconnection: Disconnection,
		Verbose:       true,
	}
	fmt.Println(server.Start())
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

func Protocol(d []byte) interface{} {
	return Data(d)
}

func Validate(id int, a []byte) (mgs.Encoder, bool) {
	return Data(a), true
}

func Disconnection(id int) {
	fmt.Println(id, "Disconnected")
}

type Data []byte

func (dt Data) Size() int {
	return len(dt)
}

func (dt Data) Encode(buff []byte) error {
	for i := range dt {
		buff[i] = dt[i]
	}
	return nil
}
