package mgs

import (
	"encoding"
	"encoding/binary"
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
	Logic        func([]*Input) []*Input
	Unmarshaler  func([]byte) encoding.BinaryMarshaler
	Validate     func([]byte) (encoding.BinaryMarshaler, bool)
	Verbose      bool

	talkers map[uint32]*talker // only Caster can write, others just read

	signal     chan uint32   // for Talkers to signal termination to the Caster.
	talkIn     chan *Input   // Data fan in from the Talkers to the Acumulator
	newTalkers chan *talker  // send new Talkers to Caster
	acToBr     chan []*Input // Acumulator to Brain
	brToCast   chan []*Input // Brain to Caster
}

func (sr *Server) Start() error {
	sr.newTalkers = make(chan *talker)
	sr.acToBr = make(chan []*Input)
	sr.brToCast = make(chan []*Input)

	sr.talkIn = make(chan *Input) // Fan In
	sr.signal = make(chan uint32)

	sr.talkers = map[uint32]*talker{}

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
	id := uint32(0)
	for {
		conn, err := listener.AcceptTCP()
		if err != nil {
			log.Print(err)
		}
		err = conn.SetReadDeadline(time.Now().Add(sr.Timeout))
		if err != nil {
			log.Print(err)
		}

		castOut := make(chan encoding.BinaryMarshaler)
		talker := talker{id, conn, 0, castOut, sr}
		sr.newTalkers <- &talker
		go talker.start()
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
				fmt.Println("acu: ", buff[:i])
			}
		}
	}
}

func (sr *Server) caster() {
	bSize := make([]byte, 2)
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
			fmt.Println("casting")
			for _, inp := range data {
				err := WriteTo(bSize, inp.T.Conn, inp.Data)
				if err != nil {
					log.Println(err)
				}
			}
		}
	}
}

func WriteTo(bSize []byte, w io.Writer, bm encoding.BinaryMarshaler) error {
	b, err := bm.MarshalBinary()
	if err != nil {
		log.Println(err)
		return err
	}
	binary.BigEndian.PutUint16(bSize, uint16(len(b)))
	dt := append(bSize, b...)
	_, err = w.Write(dt)
	fmt.Println("Wrote: ", dt)
	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}
