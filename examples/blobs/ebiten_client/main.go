package main

import (
	"flag"
	"github.com/hajimehoshi/ebiten"
	"github.com/kazhmir/gna"
	"github.com/kazhmir/gna/examples/blobs/shared"
	"image"
	_ "image/png"
	"log"
	"os"
	"time"
)

const (
	scrWid = 800
	scrHei = 600
)

var (
	D          = float64(0)
	serverAddr = flag.String("host", "localhost:8888", "Host address <ip>:<port>")
	pwd        = flag.String("pwd", "password", "Host Password")
	ball       *ebiten.Image
)

func main() {
	gna.Register(shared.Blob{}, shared.Point{}, shared.Event{}, []*shared.Blob{})
	var d float64
	ball, d = getImage("ball.png")
	ebiten.SetWindowSize(scrWid, scrHei)
	ebiten.SetWindowTitle("Blobs")
	ebiten.SetMaxTPS(30)
	game := &Game{blobs: make(map[uint64]*Blob, 16)}
	flag.Parse()

	client, player := Connect(*serverAddr, *pwd)
	game.conn = client
	game.playerID = player.ID
	game.blobs[player.ID] = &Blob{d: d, img: ball}
	D = d

	if err := ebiten.RunGame(game); err != nil {
		panic(err)
	}
}

func getImage(file string) (EImg *ebiten.Image, diameter float64) {
	reader, err := os.Open(file)
	if err != nil {
		panic(err)
	}
	defer reader.Close()
	img, _, err := image.Decode(reader)
	if err != nil {
		panic(err)
	}
	EImg = ebiten.NewImageFromImage(img)
	w, _ := EImg.Size() // image is a square
	return EImg, float64(w)
}

func Connect(addr, pwd string) (*gna.Client, *shared.Blob) {
	client, err := gna.Dial(addr)
	if err != nil {
		log.Fatal(err)
	}
	err = client.Send(pwd)
	if err != nil {
		log.Fatal(err)
	}
	dt, err := client.Recv()
	if err != nil {
		log.Fatal(err)
	}
	v, ok := dt[0].(shared.Blob)
	if ok {
		client.SetTimeout(60 * time.Second)
		client.Start()
		return client, &v
	}
	log.Fatalf("data was not blob: %v", dt[0])
	return nil, nil
}
