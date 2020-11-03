package main

import (
	"encoding"
	"fmt"
	"github.com/kazhmir/mgs"
	"time"
)

func main() {
	server := mgs.Server{
		Addr:         "localhost:8888",
		Timeout:      time.Second * 10,
		TickInterval: time.Millisecond * 50,
		Logic:        GameLogic,
		Unmarshaler:  Protocol,
		Validate:     Validate,
		Verbose:      true,
	}
	fmt.Println(server.Start())
}

func GameLogic(dt []*mgs.Input) []*mgs.Input {
	fmt.Println(dt[0].T.Ping)
	for i := range dt {
		v := dt[i].Data.(Data)
		v = append(v, 'p', 'o', 'o', 'p')
		dt[i].Data = v
	}
	return dt
}

func Protocol(d []byte) encoding.BinaryMarshaler {
	return Data(d)
}

func Validate(a []byte) (encoding.BinaryMarshaler, bool) {
	return Data(a), true
}

type Data []byte

func (dt Data) MarshalBinary() ([]byte, error) {
	return dt, nil
}
