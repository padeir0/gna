package main

import (
	"flag"
	"fmt"
	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/ebitenutil"
	"image"
	_ "image/png"
	"math"
	"os"
	//"time"
)

const (
	scrWid = 800
	scrHei = 600
	entWid = 50
	entHei = 50
)

var (
	ball       *ebiten.Image
	evilBall   *ebiten.Image
	pName      string
	toServer   chan posVec
	fromServer chan []byte
	serverAddr *string
	player     *entity
	playerIn   chan posVec
	serverOut  chan map[byte][]*message
)

func main() {
	ball = getImage("ball.png")
	evilBall = getImage("ballevil.png")
	ebiten.SetWindowSize(scrWid, scrHei)
	ebiten.SetWindowTitle("Pew Pew")
	ebiten.SetMaxTPS(30)
	game := &Game{}
	//rand.Seed(time.Now().UnixNano())

	serverAddr = flag.String("h", "localhost:8888", "Host address <ip>:<port>")
	flag.Parse()
	args := flag.Args()
	if len(args) != 1 {
		fmt.Println("NEEDS USERNAME BEEP BEEP")
		return
	}

	pIn, srvOut, err := connectTo(*serverAddr)
	if err != nil {
		panic(err)
	}
	playerIn = pIn
	serverOut = srvOut

	game.player = player
	game.player.spawn(player.x, player.y)

	if err := ebiten.RunGame(game); err != nil {
		panic(err)
	}
}

func getImage(file string) *ebiten.Image {
	reader, err := os.Open(file)
	if err != nil {
		panic(err)
	}
	defer reader.Close()
	img, _, err := image.Decode(reader)
	if err != nil {
		panic(err)
	}
	return ebiten.NewImageFromImage(img)
}
