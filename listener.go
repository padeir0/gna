package gna

import (
	"encoding/gob"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"time"
)

var (
	stdReadTimeout  = 15 * time.Second
	stdWriteTimeout = 15 * time.Second
	stdTPS          = 20 // ticks per second
)

func SetReadTimeout(d time.Duration) {
	stdReadTimeout = d
}

func SetWriteTimeout(d time.Duration) {
	stdWriteTimeout = d
}

func SetMaxTPS(tps int) {
	stdTPS = tps
}

func Register(dt ...interface{}) {
	for i := range dt {
		gob.Register(dt[i])
	}
}

func RunServer(addr string, ins *Instance) error {
	l := listener{
		mainIns: ins,
	}
	return l.start(addr)
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
	go sr.mainIns.Start()

	fmt.Println("listening on: ", addr)
	return sr.listen(chConns)
}

func (sr *listener) listen(conns chan *net.TCPConn) error {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	for {
		select {
		case conn := <-conns:
			p := newPlayer(sr.idGen.newID(), conn)
			go func() {
				sr.mainIns.world.Auth(sr.mainIns, p)
				if p.shouldStart {
					if p.grp == nil {
						p.SetInstance(sr.mainIns)
					}
					p.start()
				}
			}()
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
			log.Print(err) //TODO remove logging
			if _, ok := err.(*net.OpError); ok {
				return
			}
			continue
		}
		out <- conn
	}
}