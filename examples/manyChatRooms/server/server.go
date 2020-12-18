package server

import (
	"fmt"
	"github.com/kazhmir/gna"
	"github.com/kazhmir/gna/examples/manyChatRooms/shared"
	"log"
	"strconv"
	"sync"
	"time"
)

/*This is kinda bad and should be KILLED c:*/
type Area struct {
	ins  *gna.Instance
	room *Room
}

var rooms []*Area

func Start(addr string) error {
	gna.SetReadTimeout(60 * time.Second)
	gna.SetWriteTimeout(60 * time.Second)
	main := createMany(10)
	return gna.RunServer(addr, main)
}

func createMany(num int) (main *gna.Instance) {
	rooms = make([]*Area, num)
	for i := 0; i < num; i++ {
		rooms[i] = createInst()
	}
	return rooms[0].ins // main
}

func createInst() *Area {
	sr := &Room{Users: make(map[uint64]string, 64)}
	out := gna.NewInstance(sr)
	go out.Start()
	return &Area{out, sr}
}

type Room struct {
	Users map[uint64]string
	mu    sync.Mutex
}

func (r *Room) Update(ins *gna.Instance) {
	dt := ins.GetData()
	for _, input := range dt {
		fmt.Printf("%#v\n", input)
		switch v := input.Data.(type) {
		case string:
			r.mu.Lock()
			ins.Dispatch(ins.Players, shared.Message{Username: r.Users[input.P.ID], Data: v})
			r.mu.Unlock()
		case shared.Cmd:
			s, all := r.ExecCmd(&v, input.P)
			msg := shared.Message{Username: "server", Data: s}
			if all {
				r.mu.Lock()
				ins.Dispatch(ins.Players, msg)
				r.mu.Unlock()
				continue
			}
			ins.Dispatch(input.P, msg)
		}
	}
}

func (r *Room) Auth(ins *gna.Instance, p *gna.Player) {
	dt, err := p.Recv()
	if err != nil {
		log.Println(err)
		p.Close()
		return
	}
	if v, ok := dt.(shared.CliAuth); ok {
		r.mu.Lock()
		r.Users[p.ID] = v.Name
		fmt.Printf("%v (ID: %v) Connected.\n", v.Name, p.ID)
		ins.Dispatch(ins.Players, shared.Message{Username: "server", Data: v.Name + " Connected."})
		r.mu.Unlock()
	}
	r.mu.Lock()
	err = p.Send(shared.SrAuth{UserID: p.ID})
	r.mu.Unlock()
	if err != nil {
		log.Println(err)
		p.Close()
	}
}

func (r *Room) Disconn(ins *gna.Instance, p *gna.Player) {
	fmt.Printf("%v (ID: %v) Disconnected. Reason: %v\n", r.Users[p.ID], p.ID, p.Error())
	ins.Dispatch(ins.Players, shared.Message{Username: "server", Data: r.Users[p.ID] + " Disconnected."})
	r.mu.Lock()
	delete(r.Users, p.ID)
	r.mu.Unlock()
}

func (r *Room) ExecCmd(cmd *shared.Cmd, p *gna.Player) (msg string, toAll bool) {
	switch cmd.T {
	case shared.CName:
		r.mu.Lock()
		old := r.Users[p.ID]
		r.Users[p.ID] = cmd.Data
		r.mu.Unlock()
		return fmt.Sprintf("%v changed name to %v", old, cmd.Data), true
	case shared.CRoom:
		n, err := strconv.Atoi(cmd.Data)
		if err != nil {
			return fmt.Sprintf("could not change room: %v", err), false
		}
		if n > len(rooms) || n < 0 {
			return fmt.Sprintf("could not change room: there is no room %v", n), false
		}
		r.mu.Lock()
		name := r.Users[p.ID]
		delete(r.Users, p.ID)
		r.mu.Unlock()
		p.SetInstance(rooms[n].ins)
		rooms[n].room.mu.Lock()
		rooms[n].room.Users[p.ID] = name
		rooms[n].room.mu.Unlock()
		return fmt.Sprintf("%v changed to room %v", name, n), true
	case shared.Num:
		r.mu.Lock()
		n := len(r.Users)
		r.mu.Unlock()
		return fmt.Sprintf("People in room: %v", n), false
	}
	return "invalid command", false
}
