package mgs

import (
	"encoding"
	"fmt"
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
	Logic        func([]*Input) map[int]io.WriterTo
	Unmarshaler  func([]byte) encoding.BinaryMarshaler
	Validate     func([]byte) (io.WriterTo, bool)
	Verbose      bool

	talkers map[int]*talker // only Caster can write, others just read

	signal     chan int                 // for Talkers to signal termination to the Caster.
	talkIn     chan *Input              // Data fan in from the Talkers to the Acumulator
	newTalkers chan *talker             // send new Talkers to Caster
	acToBr     chan []*Input            // Acumulator to Brain
	brToCast   chan map[int]io.WriterTo // Brain to Caster
}

func (sr *Server) Start() error {
	sr.newTalkers = make(chan *talker)
	sr.acToBr = make(chan []*Input)
	sr.brToCast = make(chan map[int]io.WriterTo)

	sr.talkIn = make(chan *Input) // Fan In
	sr.signal = make(chan int)

	sr.talkers = map[int]*talker{}

	go sr.brain()
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
	if sr.Verbose {
		fmt.Println("Listening at:", sr.Addr)
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

		castOut := make(chan io.WriterTo, 1)
		talker := talker{id, conn, castOut, sr}
		sr.newTalkers <- &talker
		go talker.Talk()
		id++
	}
}

func (sr *Server) brain() {
	for {
		sr.brToCast <- sr.Logic(<-sr.acToBr)
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
			if sr.Verbose {
				fmt.Println(msg)
			}
		}
	}
}

func (sr *Server) caster() {
	for {
		select {
		case id := <-sr.signal:
			delete(sr.talkers, id)
			if sr.Verbose {
				fmt.Println("Terminating: ", id)
			}
		case newTalker := <-sr.newTalkers:
			sr.talkers[newTalker.Id] = newTalker
			if sr.Verbose {
				fmt.Println("New Talker: ", newTalker.Id)
			}
		case data := <-sr.brToCast:
			for id, w := range data {
				if v, ok := sr.talkers[id]; ok {
					v.castOut <- w
				}
			}
		}
	}
}
