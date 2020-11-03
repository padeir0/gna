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

func GameLogic(dt []*mgs.Input) map[uint32][]encoding.BinaryMarshaler {
	out := map[uint32][]encoding.BinaryMarshaler{}
	for i := range dt {
		v := dt[i].Data.(Data)
		if _, ok := out[dt[i].T.Id]; ok {
			out[dt[i].T.Id] = append(out[dt[i].T.Id], v)
		} else {
			out[dt[i].T.Id] = []encoding.BinaryMarshaler{v}
		}
	}
	time.Sleep(time.Millisecond * 40) // simulating a load
	return out
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
