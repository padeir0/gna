package mgs

import (
	"encoding"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type Server struct {
	// Read only fields
	Addr         string
	Timeout      time.Duration
	TickInterval time.Duration
	Logic        func([]*Input) map[uint32][]encoding.BinaryMarshaler
	Unmarshaler  func([]byte) encoding.BinaryMarshaler
	Validate     func([]byte) (encoding.BinaryMarshaler, bool)
	Verbose      bool

	talkers map[uint32]*talker // only Caster can write, others just read

	signal     chan uint32                                // for Talkers to signal termination to the Caster.
	talkIn     chan *Input                                // Data fan in from the Talkers to the Acumulator
	newTalkers chan *talker                               // send new Talkers to Caster
	acToBr     chan []*Input                              // Acumulator to Brain
	brToDisp   chan map[uint32][]encoding.BinaryMarshaler // Brain to Dispatcher
}

func (sr *Server) Start() error {
	sr.newTalkers = make(chan *talker)
	sr.acToBr = make(chan []*Input)
	sr.brToDisp = make(chan map[uint32][]encoding.BinaryMarshaler)

	sr.talkIn = make(chan *Input) // Fan In
	sr.signal = make(chan uint32)

	sr.talkers = map[uint32]*talker{}

	go sr.brain()
	go sr.dispatcher()
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
		case newTalker := <-sr.newTalkers:
			sr.talkers[newTalker.Id] = newTalker
			if sr.Verbose {
				fmt.Println("New Talker: ", newTalker.Id)
			}
		/* Create Workers to handle the writes, separate the writes by id
		that way an entire array of writes can be negated if an user disconnects.
		something like map[uint32][]*Input
		*/
		case data := <-sr.brToDisp:
			var ok bool
			var t *talker
			for id, marshSlice := range data {
				for _, marsh := range marshSlice {
					if t, ok = sr.talkers[id]; !ok {
						log.Println("Cancelling packets to: ", id, ". Talker Terminated.")
						break
					}
					err := WriteTo(t.conn, marsh)
					if err != nil {
						if errors.Is(err, syscall.EPIPE) {
							log.Println("Cancelling packets to: ", id, ".", err)
							break
						}
						log.Println(err)
					}
				}
			}
		}
	}
}

func WriteTo(w io.Writer, bm encoding.BinaryMarshaler) error {
	bSize := make([]byte, 2)
	b, err := bm.MarshalBinary()
	if err != nil {
		log.Println(err)
		return err
	}
	binary.BigEndian.PutUint16(bSize, uint16(len(b)))
	dt := append(bSize, b...)
	_, err = w.Write(dt)
	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}
