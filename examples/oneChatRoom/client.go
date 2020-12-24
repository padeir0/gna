package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/kazhmir/gna"
)

func CliUpdate() {
	ticker := time.NewTicker(50 * time.Millisecond)
	for {
		<-ticker.C
		data := cli.RecvBatch() // err is aways nil with cli.Start()
		if err := cli.Error(); err != nil {
			log.Fatalf("Recv Error: %v\n", err)
		}
		if len(data) == 0 {
			continue
		}
		for _, dt := range data {
			switch v := dt.(type) {
			case Message:
				fmt.Printf("%v: %v\n", v.Username, v.Data)
			default:
				fmt.Printf("\n%#v, %T\n", dt, dt)
			}
		}
	}
}

func ClientLoop() {
	for {
		msg, err := term.ReadString('\n')
		if err != nil {
			fmt.Println(err)
			continue
		}
		cli.Dispatch(strings.ReplaceAll(msg, "\n", ""))
		if err := cli.Error(); err != nil {
			log.Fatalf("Send Error: %v\n", err)
		}
	}
}

func Connect(addr, name string) *gna.Client {
	cli, err := gna.Dial(addr)
	if err != nil {
		log.Fatal(err)
	}
	err = cli.Send(cliAuth{name})
	if err != nil {
		log.Fatal(err)
	}
	dts, err := cli.Recv()
	if v, ok := dts.(srAuth); ok {
		fmt.Printf("UserID: %v\n", v.UserID)
	}
	cli.Start()
	return cli
}
