package main

import (
	"flag"
	"fmt"
	"github.com/kazhmir/mgs"
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

	server := &EchoServer{}
	ins := mgs.NewInstance(server)
	if err := mgs.RunServer("localhost:8888", ins); err != nil {
		log.Fatal(err)
	}
}

type EchoServer struct {
}

func (es *EchoServer) Update(ins *mgs.Instance) {
	dt := ins.GetData()
	for i := range dt {
		ins.Unicast(dt[i].P, dt[i].Data)
	}
}

func (es *EchoServer) Auth(p *mgs.Player) {
	fmt.Println("Connected: ", p.ID)
}

func (es *EchoServer) Disconn(p *mgs.Player) {
	fmt.Println("Disconnected: ", p.ID)
}
