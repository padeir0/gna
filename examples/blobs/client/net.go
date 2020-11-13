package networking

import (
	"net"
	"sync"
)

type Talker struct {
	conn net.Conn

	Read  chan interface{}
	Write chan string
	Term  chan error

	buff []byte
	i, n int

	mu   sync.Mutex
	dead bool
}

func (t *Talker) Terminate() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if !t.dead {
		t.conn.Close()
		t.Read.Close()
		t.Write.Close()
	}
}

func (t *Talker) Start() {
}

func (t *Talker) ear() {
}

func (t *Talker) mouth() {
}
