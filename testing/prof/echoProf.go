package main

import (
	"flag"
	"fmt"
	"github.com/kazhmir/gna"
	"log"
	"os"
	"runtime/pprof"
)

var cpuProfile = flag.String("ppcpu", "", "Write cpu profile to file")

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

	if err := gna.RunServer("localhost:8888", &EchoServer{}); err != nil {
		log.Fatal(err)
	}
}

type EchoServer struct {
	gna.Net
}

func (es *EchoServer) Update() {
	dt := es.GetData()
	for i := range dt {
		es.Dispatch(dt[i].P, dt[i].Data)
	}
}

func (es *EchoServer) Auth(p *gna.Player) {
	fmt.Println("Connected: ", p.ID)
}

func (es *EchoServer) Disconn(p *gna.Player) {
	fmt.Printf("Disconnected: %v, Reason: %v\n", p.ID, p.Error())
}
