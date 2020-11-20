package main

import (
	"encoding/binary"
	"errors"
	"io"
	"log"
	"math"
	"net"
)

// the _ is just to remember its a package type
const (
	logged_ = iota
	fire_
	mov_
	dmg_
	health_
	spawn_
	accepted_
)

var (
	errWrongServerPkg error = errors.New("Server sent wrong package size.")
)

type posVec struct {
	x, y, rot float32
	tp, id    byte
}

func (pv posVec) toByte() []byte {
	buff := make([]byte, 14)
	buff[0] = pv.tp
	buff[1] = pv.id
	binary.BigEndian.PutUint32(buff[2:], math.Float32bits(pv.x))
	binary.BigEndian.PutUint32(buff[6:], math.Float32bits(pv.y))
	binary.BigEndian.PutUint32(buff[10:], math.Float32bits(pv.rot))
	return buff
}

func (pv posVec) writeTo(w io.Writer) (int64, error) {
	n, err := w.Write(pv.toByte())
	return int64(n), err
}

func pvFromByte(buff []byte) posVec {
	pv := posVec{}
	pv.tp = buff[0]
	pv.id = buff[1]
	pv.x = math.Float32frombits(binary.BigEndian.Uint32(buff[2:6]))
	pv.y = math.Float32frombits(binary.BigEndian.Uint32(buff[6:10]))
	pv.rot = math.Float32frombits(binary.BigEndian.Uint32(buff[10:]))
	return pv
}

type talker struct {
}

type message struct {
	tp   byte
	data []byte
}

func connectTo(addr string) (chan posVec, chan map[byte][]*message, error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, nil, err
	}
	defer conn.Close()
	buff := make([]byte, 14)
	_, err = conn.Write(append([]byte{logged_}, []byte(pName)...))
	if err != nil {
		return nil, nil, err
	}
	n, err := conn.Read(buff)
	if err != nil {
		return nil, nil, err
	}
	if n != 14 {
		return nil, nil, errWrongServerPkg
	}
	pv := pvFromByte(buff)
	player = &entity{
		x:   float64(pv.x),
		y:   float64(pv.y),
		rot: float64(pv.rot),
		img: ball,
	}
	playerIn := make(chan posVec)
	serverOut := make(chan map[byte][]*message)
	go func() {
		for {
			select {
			case input <- playerIn:
				input.writeTo(conn)
			default:
				n, err = conn.Read(buff)
				if err != nil {
					log.Print(err)
				}
				if n > 1 {
					perPlayer := map[byte][]*message{}
					for n > 0 {
						if val, ok := perPlayer[buff[1]]; ok {
							perPlayer[buff[1]] = append(val, &message{buff[0], buff[1:]})
						} else {
							perPlayer[buff[1]] = []message{{buff[0], buff[1:]}}
						}
						n, err := conn.Read(buff)
					}
					serverOut <- perPlayer
				}
			}
		}
	}()
	return playerIn, serverOut, nil
}
