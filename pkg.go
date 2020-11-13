package mgs

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"time"
)

type Server struct {
	// Read only fields
	Addr          string
	Timeout       time.Duration // default 5 s
	TickInterval  time.Duration // default 50 ms
	Logic         func([]*Input) map[uint32][]Encoder
	Unmarshaler   func([]byte) Encoder
	Validate      func(int, []byte) (Encoder, bool)
	Disconnection func(int) // default: does nothing
	Verbose       bool

	talkers map[uint32]*talker // only Dispatcher can modify, others just read

	signal     chan uint32               // for Talkers to signal termination to the Caster.
	talkIn     chan *Input               // Data fan in from the Talkers to the Acumulator
	newTalkers chan *talker              // send new Talkers to Caster
	acToBr     chan []*Input             // Acumulator to Brain
	brToDisp   chan map[uint32][]Encoder // Brain to Dispatcher
}

func (sr *Server) Start() error {
	sr.fillDefault()

	sr.newTalkers = make(chan *talker)
	sr.acToBr = make(chan []*Input)
	sr.brToDisp = make(chan map[uint32][]Encoder)
	sr.talkIn = make(chan *Input) // Fan In
	sr.signal = make(chan uint32)
	sr.talkers = map[uint32]*talker{}

	go sr.brain()
	go sr.dispatcher()
	go sr.acumulator()

	return sr.listen()
}

func (sr *Server) fillDefault() {
	if sr.Addr == "" {
		sr.Addr = "0.0.0.0:27272"
	}
	if sr.Timeout == 0 {
		sr.Timeout = time.Second * 5
	}
	if sr.TickInterval == 0 {
		sr.TickInterval = time.Millisecond * 50
	}
	if sr.Logic == nil {
		panic("Server.Logic cannot be nil!")
	}
	if sr.Unmarshaler == nil {
		panic("Server.Unmarshaler cannot be nil!")
	}
	if sr.Validate == nil {
		panic("Server.Validate cannot be nil!")
	}
	if sr.Disconnection == nil {
		sr.Disconnection = func(int) {} // dummy
	}
}

func (sr *Server) listen() error {
	addr, err := net.ResolveTCPAddr("tcp", sr.Addr)
	listener, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return err
	}
	if sr.Verbose {
		fmt.Println("Listening at:", sr.Addr)
	}

	defer listener.Close()
	id := uint32(0)
	chConns := make(chan *net.TCPConn)
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	go func() {
		for {
			conn, err := listener.AcceptTCP()
			if err != nil {
				log.Print(err)
				continue
			}
			chConns <- conn
		}
	}()
	for {
		select {
		case conn := <-chConns:
			talker := talker{
				Id:   id,
				conn: conn,
				sr:   sr,
			}
			sr.newTalkers <- &talker
			go talker.start()
			id++
		case <-sig:
			fmt.Println("Stopping server...")
			for _, t := range sr.talkers {
				t.Terminate()
			}
			return nil
		}
	}
}

func (sr *Server) brain() {
	for {
		sr.brToDisp <- sr.Logic(<-sr.acToBr)
	}
}

func (sr *Server) acumulator() {
	ticker := time.NewTicker(sr.TickInterval)
	defer ticker.Stop()
	buff := make([]*Input, 1000)
	var i int
	for {
		select {
		case <-ticker.C:
			if i > 0 {
				sr.acToBr <- buff[:i]
				i = 0
			}
		case msg := <-sr.talkIn:
			if i >= len(buff) {
				buff = append(buff, make([]*Input, 50)...)
			}
			buff[i] = msg
			i++
		}
	}
}

func (sr *Server) dispatcher() {
	for {
		select {
		case id := <-sr.signal:
			delete(sr.talkers, id)
			if sr.Verbose {
				fmt.Println("Terminating: ", id)
			}
			continue // this may be useless
		case newTalker := <-sr.newTalkers:
			sr.talkers[newTalker.Id] = newTalker
			if sr.Verbose {
				fmt.Println("New Talker: ", newTalker.Id)
			}
		/* Create Workers to handle the writes*/
		case data := <-sr.brToDisp:
			var ok bool
			var t *talker
			c := make(chan struct{})
			for id, marshSlice := range data {
				if t, ok = sr.talkers[id]; !ok {
					log.Println("Talker Terminated. Cancelling packets to: ", id)
					continue
				}
				t.mouthSig <- c
				t.mouthDt <- marshSlice
			}
			close(c)
		}
	}
}
