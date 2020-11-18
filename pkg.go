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

/*Server is a struct that defines properties of the underlying TCP server.
 */
type Server struct {
	// Read only fields

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
	Validate func(int, *Packet) (Encoder, bool)
	/*Disconnection receives the ID of the talker that is terminating
	before the connection is closed. Default: does nothing
	*/
	Disconnection func(int)
	Verbose       bool // should become Debug and do more things.
	MaxPlayers    int

	talkers map[uint64]*talker // only Dispatcher can modify, others just read

	signal     chan uint64               // for Talkers to signal termination to the Caster.
	talkIn     chan *Input               // Data fan in from the Talkers to the Acumulator
	newTalkers chan *talker              // send new Talkers to Caster
	acToBr     chan []*Input             // Acumulator to Brain
	brToDisp   chan map[Sender][]Encoder // Brain to Dispatcher
}

/*Start setups the server and starts it*/
func (sr *Server) Start() error {
	sr.fillDefault()

	sr.newTalkers = make(chan *talker)
	sr.acToBr = make(chan []*Input)
	sr.brToDisp = make(chan map[Sender][]Encoder)
	sr.talkIn = make(chan *Input) // Fan In
	sr.signal = make(chan uint64)
	sr.talkers = map[uint64]*talker{}

	go sr.brain()
	go sr.dispatcher()
	go sr.acumulator()

	return sr.listen()
}

/*AllTalkers returns a *Group with all the talkers currently running*/
func (sr *Server) AllTalkers() *Group {
	return &Group{tMap: sr.talkers}
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
	id := uint64(0)
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
			if len(sr.talkers) >= sr.MaxPlayers {
				//writeTo(conn, &[]byte{}, ErrServerFull)
				conn.Close()
				break
			}
			talker := talker{
				ID:   id,
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
				b := make([]*Input, i)
				copy(b, buff[:i])
				sr.acToBr <- b
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
