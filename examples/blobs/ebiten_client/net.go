package main

import (
	"github.com/kazhmir/gna"
	"github.com/kazhmir/gna/examples/blobs/shared"
	"log"
)

func Connect(addr, pwd string) (*gna.Client, *shared.Blob) {
	client, err := gna.Dial(addr)
	if err != nil {
		log.Fatal(err)
	}
	err = client.Send(shared.Auth{pwd})
	if err != nil {
		log.Fatal(err)
	}
	dt, err := client.Recv()
	if err != nil {
		log.Fatal(err)
	}
	v, ok := dt[0].(shared.Blob)
	if ok {
		return client, &v
	}
	panic("data was not shared.Blob")
}
