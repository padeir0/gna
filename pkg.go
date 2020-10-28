package mgs

import (
	"io"
	"log"
	"net"
	"time"
)

type Server struct {
	// Read only fields
	Addr         string
	Timeout      time.Duration
	TickInterval time.Duration
	Brain        func(cAcumu <-chan []*Input, cCaster chan<- io.WriterTo)
	Protocol     func([]byte) interface{}
	Validate     func([]byte) bool

	talkers map[int]*talker // only Caster can write, others just read

	signal     chan int         // for Talkers to signal termination to the Caster.
	talkIn     chan *Input      // Data fan in from the Talkers to the Acumulator
	newTalkers chan *talker     // send new Talkers to Caster
	acToBr     chan []*Input    // Acumulator to Brain
	brToCast   chan io.WriterTo // Brain to Caster
}

func (sr *Server) Start() error {
	sr.newTalkers = make(chan *talker)
	sr.acToBr = make(chan []*Input)
	sr.brToCast = make(chan io.WriterTo)

	sr.talkIn = make(chan *Input) // Fan In
	sr.signal = make(chan int)

	go sr.Brain(sr.acToBr, sr.brToCast)
	go sr.caster()
	go sr.acumulator()

	return sr.listen()
}

func (sr *Server) listen() error {
	addr, err := net.ResolveTCPAddr("tcp", sr.Addr)
	listener, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return err
	}

	defer listener.Close()
	id := 0
	for {
		conn, err := listener.AcceptTCP()
		if err != nil {
			log.Print(err)
		}
		err = conn.SetReadDeadline(time.Now().Add(sr.Timeout))
		if err != nil {
			log.Print(err)
		}

		castOut := make(chan io.WriterTo, 2)
		talker := talker{id, conn, castOut, sr}
		sr.newTalkers <- &talker
		go talker.Talk()
		id++
	}
}

func (sr *Server) acumulator() {
	ticker := time.NewTicker(time.Millisecond * sr.TickInterval)
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

func (sr *Server) caster() {
	for {
		select {
		case id := <-sr.signal:
			delete(sr.talkers, id)
		case newTalker := <-sr.newTalkers:
			sr.talkers[newTalker.id] = newTalker
		case data := <-sr.brToCast:
			for id := range sr.talkers {
				sr.talkers[id].castOut <- data
			}
		}
	}
}
