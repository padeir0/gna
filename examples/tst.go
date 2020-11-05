package main

import (
	"encoding"
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
		Addr:         "localhost:8888",
		Timeout:      time.Second * 10,
		TickInterval: time.Millisecond * 100,
		Logic:        GameLogic,
		Unmarshaler:  Protocol,
		Validate:     Validate,
		Verbose:      true,
	}
	fmt.Println(server.Start())
}

var iter int

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
