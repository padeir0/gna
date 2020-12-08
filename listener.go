package mgs

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"time"
)

var (
	stdTimeout = 5 * time.Second
	stdTPS     = 20 // 20 times per second
)

func SetStdTimeout(d time.Duration) {
	stdTimeout = d
}

func SetStdTPS(tps int) {
	stdTPS = tps
}

func RunServer(addr string, ins *Instance) error {
	ins.Start()
	l := listener{
		mainIns: ins,
	}
	return l.start(addr)
}

type Server interface {
	Update(*Instance)
	Disconn(*Player)
	Auth(*Player)
}

type listener struct {
	mainIns *Instance
	idGen   id
}

/*start setups the server and starts it*/
func (sr *listener) start(addr string) error {
	tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
	listener, err := net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		return err
	}
	defer listener.Close()
	chConns := make(chan *net.TCPConn)

	go connRecv(listener, chConns)

	return sr.listen(chConns)
}

func (sr *listener) listen(conns chan *net.TCPConn) error {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	for {
		select {
		case conn := <-conns:
			p := newPlayer(sr.idGen.newID(), conn)
			sr.mainIns.handler.Auth(p)
			if !p.dead {
				if p.ins == nil {
					p.SetInstance(sr.mainIns)
				}
				go p.start()
			}
		case <-sig:
			fmt.Println("Stopping server...")
			sr.mainIns.terminate()
			return nil
		}
	}
}

func connRecv(l *net.TCPListener, out chan *net.TCPConn) {
	for {
		conn, err := l.AcceptTCP()
		if err != nil {
			log.Print(err)
			if _, ok := err.(*net.OpError); ok {
				return
			}
			continue
		}
		out <- conn
	}
}
