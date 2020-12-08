package main

import (
	"errors"
	"github.com/kazhmir/mgs"
	"github.com/kazhmir/mgs/examples/blobs/shared"
	"log"
	"sync"
	"time"
)

func main() {
	server := Server{
		make(map[uint64]*shared.Blob, 64),
		500,
		&EventList{list: make([]*shared.Event, 128)},
		sync.Mutex{},
	}
	mgs.SetStdTimeout(5 * time.Second)
	mgs.SetStdTPS(20) // 20 ticks per second. It's the rate at which you process user data.
	ins := mgs.NewInstance(&server)
	if err := mgs.RunServer("0.0.0.0:8888", ins); err != nil {
		log.Fatal(err) // if the user uses SIGTERM, the server tries to stop without errors.
	}
	/*
	   to create new instances:

	   ins := mgs.NewInstance(Server)
	   Player.SetInstance(ins)

	   if done in the Auth method it can serve as load balancing, if the Auth doesn't set the instance
	   the default is the main instance of the server.

	   instances can be set inside Server.Update too

	   If the given instance is not running, the package will return an error or panic?
	*/
}

type Server struct {
	blobs map[uint64]*shared.Blob
	size  int
	evl   *EventList
	mu    sync.Mutex
}

/* The following functions do not send messages through channels,
instead they only append the data into a buffer with the necessary
information to dispatch it to the receivers

disp.Broadcast(out...) // Mark data to Send to all talkers
disp.Multicast(mgs.Group, out...) // Mark data to Send to a group of talkers
disp.Unicast(mgs.Conn, out...) // Mark data to Send to a single talker

the disp.Dispatch() then sends the messages through channels,
and signals for the work to start, similar to older implementation
*/
func (sr *Server) Update(ins *mgs.Instance) {
	sr.mu.Lock()
	defer sr.mu.Unlock()
	ins.Broadcast(sr.Events()...) // the client will receive this as a bunch of *shared.Events
	data := ins.GetData()
	var n int
	for _, input := range data {
		v, _ := input.Data.(string)
		b := sr.blobs[input.P.ID]
		for j := range v {
			switch v[j] {
			case 'w':
				b.Move(false)
			case 's':
				b.Move(true)
			case 'a':
				b.Rotate(false)
			case 'd':
				b.Rotate(true)
			}
		}
		ins.Broadcast(b)
		n++
	}
}

/*It may not be safe to send messages to the conn at this time
since it may terminate due to client disconnection or crash.
Make the Dispatcher handle this instead of each talker goroutine?
*/
func (sr *Server) Disconn(p *mgs.Player) {
	sr.RmBlob(p.ID)
}

func (sr *Server) Auth(p *mgs.Player) {
	var a shared.Auth
	err := p.Recv(&a)
	if err != nil {
		log.Println(err)
		p.Terminate()
	}
	if a.Pwd == "password" {
		p.Send(sr.NewBlob(p.ID))
	}
	p.Send(errors.New("invalid password"))
	p.Terminate()
}

func (gm *Server) NewBlob(id uint64) *shared.Blob {
	b := &shared.Blob{}
	gm.mu.Lock()
	gm.blobs[id] = b
	gm.evl.Add(&shared.Event{ID: id, T: shared.EBorn})
	gm.mu.Unlock()
	return b.Spawn(gm.size)
}

func (gm *Server) RmBlob(id uint64) {
	gm.mu.Lock()
	delete(gm.blobs, id)
	gm.evl.Add(&shared.Event{ID: id, T: shared.EDied})
	gm.mu.Unlock()
}

func (gm *Server) Events() []interface{} {
	return gm.evl.Consume()
}

type EventList struct {
	list []*shared.Event
	i    int
}

func (evl *EventList) Add(e *shared.Event) {
	if evl.i >= len(evl.list) {
		evl.list = append(evl.list, make([]*shared.Event, 64)...)
	}
	evl.list[evl.i] = e
	evl.i++
}

func (evl *EventList) Consume() []interface{} {
	out := make([]interface{}, evl.i)
	for i := 0; i < evl.i; i++ {
		out[i] = evl.list[i]
	}
	evl.i = 0
	return out
}
