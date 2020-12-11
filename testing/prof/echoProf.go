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

	server := &EchoServer{}
	ins := gna.NewInstance(server)
	if err := gna.RunServer("localhost:8888", ins); err != nil {
		log.Fatal(err)
	}
}

type EchoServer struct {
}

func (es *EchoServer) Update(ins *gna.Instance) {
	dt := ins.GetData()
	for i := range dt {
		ins.Unicast(dt[i].P, dt[i].Data)
	}
}

func (es *EchoServer) Auth(ins *gna.Instance, p *gna.Player) {
	fmt.Println("Connected: ", p.ID)
}

func (es *EchoServer) Disconn(ins *gna.Instance, p *gna.Player) {
	fmt.Printf("Disconnected: %v, Reason: %v\n", p.ID, p.Error())
}
