package mgs

import (
	"fmt"
	"log"
	"math"
	"net"
	"os"
	"os/signal"
	"time"
)

/*NewTalker returns a talker that will receive and send messages
in separate goroutines. talker.Start() itself should be called in a different goroutine.
*/
func NewTalker(id uint64, conn net.Conn, ctx *Context) *Talker {
	return &Talker{
		ID:   id,
		conn: conn,
		ctx:  ctx,
	}
}

/*Server is a struct that defines properties of the underlying TCP server.
Except otherwise noted, all fields are readonly.
*/
type Server struct {
	/*Server address in the form <ip>:<port> */
	Addr string
	/*Maximum time a client can be idle. Default: 5s*/
	Timeout time.Duration
	/*Interval between server ticks. Default: 50ms*/
	TickInterval time.Duration
	/*Logic is the game logic that handles inputs and returns the proper responses*/
	Logic func([]*Input) map[Sender][]Encoder
	/*Unmarshaler receives a byte packet and returns the proper object
	to be used in the game logic*/
	Unmarshaler func(*Packet) interface{}
	/*Validate is receives the first packet from the client
	and returns a response and if the client is accepted
	*/
	Validate func(uint64, *Packet) (Encoder, bool)
	/*Disconnection receives the ID of the talker that is terminating
	before the connection is closed. Default: does nothing
	*/
	Disconnection func(uint64)
	Verbose       bool // should become Debug and do more things.
	MaxPlayers    int

	ctx     *Context
	talkers map[uint64]*Talker // only Dispatcher can write, others just read

	newTalkers chan *Talker              // send new Talkers to Caster
	acToBr     chan []*Input             // Acumulator to Brain
	brToDisp   chan map[Sender][]Encoder // Brain to Dispatcher

	nextID id
}

/*Start setups the server and starts it*/
func (sr *Server) Start() error {
	sr.fillDefault()

	sr.newTalkers = make(chan *Talker)
	sr.acToBr = make(chan []*Input)
	sr.brToDisp = make(chan map[Sender][]Encoder)
	sr.talkers = map[uint64]*Talker{}

	sr.ctx = &Context{
		Out:         make(chan *Input),
		TermSig:     make(chan uint64),
		Timeout:     sr.Timeout,
		Discon:      sr.Disconnection,
		Unmarshaler: sr.Unmarshaler,
		Validate:    sr.Validate,
	}

	go sr.brain()
	go sr.dispatcher()
	go sr.acumulator()

	return sr.listen()
}

/*AllTalkers returns a *Group with all the talkers currently running*/
func (sr *Server) AllTalkers() *Group {
	return &Group{tMap: sr.talkers}
}

/*AddTalker adds the talker to the dispatcher list*/
func (sr *Server) AddTalker(t *Talker) {
	sr.newTalkers <- t
}

/*NewTalkerID returns an incremental uint64 ID.
This means you may run into problems if your server receives (2^64)-1 connections
and the first few are still connected.
This means more than 18 quintillion connections without a restart. Or in other terms,
73 years if every single human alive sucessfully connects every second,
not counting population growth of course.
*/
func (sr *Server) NewTalkerID() uint64 {
	return sr.nextID.newID()
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
		sr.Disconnection = func(uint64) {} // dummy
	}
	if sr.MaxPlayers <= 0 {
		sr.MaxPlayers = math.MaxInt64
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
	chConns := make(chan *net.TCPConn)
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	go func() {
		for {
			conn, err := listener.AcceptTCP()
			if err != nil {
				log.Print(err)
				if _, ok := err.(*net.OpError); ok {
					return
				}
				continue
			}
			chConns <- conn
		}
	}()
	for {
		select {
		case conn := <-chConns:
			// racy but doesn't matter in this context
			if len(sr.talkers) >= sr.MaxPlayers {
				//writeTo(conn, &[]byte{}, ErrServerFull)
				conn.Close()
				break
			}
			t := NewTalker(sr.NewTalkerID(), conn, sr.ctx)
			sr.AddTalker(t)
			go t.Start()
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
				b := make([]*Input, i)
				copy(b, buff[:i])
				sr.acToBr <- b
				i = 0
			}
		case msg := <-sr.ctx.Out:
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
		case id := <-sr.ctx.TermSig:
			delete(sr.talkers, id)
			if sr.Verbose {
				fmt.Println("Terminating: ", id)
			}
			continue // this may be useless
		case newTalker := <-sr.newTalkers:
			sr.talkers[newTalker.ID] = newTalker
			if sr.Verbose {
				fmt.Println("New Talker: ", newTalker.ID)
			}
		/* Create Workers to handle the writes*/
		case data := <-sr.brToDisp:
			c := make(chan struct{})
			for send, encs := range data {
				if send.Rectify(&sr.talkers) {
					send.Send(c, encs)
				}
			}
			close(c)
		}
	}
}
