package main

import (
	"flag"
	"fmt"
	"github.com/kazhmir/mgs"
	"os"
	"runtime/pprof"
	"time"
)

var cpuProfile = flag.String("ppcpu", "", "Write cpu profile to file")
var memProfile = flag.String("ppmem", "", "Write mem profile to file")
var goProfile = flag.String("ppgo", "", "Write goroutine profile to file")
var memFile *os.File
var goFile *os.File

func main() {
	flag.Parse()
	if *cpuProfile != "" {
		f, err := os.Create(*cpuProfile)
		if err != nil {
			panic(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}
	if *memProfile != "" {
		f, err := os.Create(*memProfile)
		if err != nil {
			panic(err)
		}
		memFile = f
		defer f.Close()
	}
	if *goProfile != "" {
		f, err := os.Create(*goProfile)
		if err != nil {
			panic(err)
		}
		goFile = f
		defer f.Close()
	}

	server := mgs.Server{
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

var iter int

func GameLogic(dt []*mgs.Input) map[mgs.Sender][]mgs.Encoder {
	out := map[mgs.Sender][]mgs.Encoder{}
	for i := range dt {
		v := dt[i].Data.(Data)
		if _, ok := out[dt[i].T]; ok {
			out[dt[i].T] = append(out[dt[i].T], v)
		} else {
			out[dt[i].T] = []mgs.Encoder{v}
		}
	}
	if iter == 5 {
		if memFile != nil {
			err := pprof.Lookup("heap").WriteTo(memFile, 0)
			if err != nil {
				fmt.Println(err)
			}
			memFile = nil
		}
		if goFile != nil {
			err := pprof.Lookup("goroutine").WriteTo(goFile, 0)
			if err != nil {
				fmt.Println(err)
			}
			goFile = nil
		}
		iter = 0
	}
	iter++
	return out
}

func Protocol(d []byte) mgs.Encoder {
	return Data(d)
}

func Validate(id int, a []byte) (mgs.Encoder, bool) {
	return Data(a), true
}

func Disconnection(id int) {
	fmt.Println(id, "Disconnected")
}

type Data []byte

func (dt Data) MarshalBinary() ([]byte, error) {
	return dt, nil
}

func (dt Data) Size() int {
	return len(dt)
}

func (dt Data) Encode(buff []byte) error {
	for i := range dt {
		buff[i] = dt[i]
	}
	return nil
}
