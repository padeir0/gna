package main

import (
	"flag"
	"fmt"
	"github.com/hajimehoshi/ebiten"
	"image"
	_ "image/png"
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
)

func main() {
	ball, d := getImage("ball.png")
	ebiten.SetWindowSize(scrWid, scrHei)
	ebiten.SetWindowTitle("Blobs")
	ebiten.SetMaxTPS(30)
	game := &Game{blobs: make(map[uint64]*Blob, 16)}
	flag.Parse()

	client, player := Connect(*serverAddr, *pwd)
	game.conn = client
	game.playerID = player.ID
	game.blobs[player.ID] = player
	game.playerID = 0
	game.blobs[0] = &Blob{d: d, img: ball}
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
