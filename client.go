package gna

import (
	"encoding/gob"
	"fmt"
	"net"
	"time"
)

func Dial(addr string) (*Client, error) {
	c, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}
	cli := &Client{
		p: &Player{
			conn:        c,
			enc:         gob.NewEncoder(c),
			dec:         gob.NewDecoder(c),
			rTimeout:    stdReadTimeout,
			wTimeout:    stdWriteTimeout,
			cDisp:       make(chan interface{}),
			dc:          make(chan *Player, 2),
			acu:         &acumulator{dt: make([]*Input, 64)},
			shouldStart: true,
		},
	}
	return cli, nil
}

type Client struct {
	p       *Player
	started bool
}

func (c *Client) Send(dt interface{}) error {
	if c.started {
		c.p.cDisp <- dt
		return nil
	}
	err := c.p.conn.SetWriteDeadline(time.Now().Add(c.p.wTimeout * time.Second))
	if err != nil {
		return err
	}
	err = c.p.Send(dt)
	if err != nil {
		err = fmt.Errorf("%w while encoding: %v", err, dt)
	}
	return err
}

func (c *Client) Recv() ([]interface{}, error) {
	if c.started {
		dt := c.p.acu.consume()
		out := make([]interface{}, len(dt))
		for i := range dt {
			out[i] = dt[i].Data
		}
		return out, nil
	}
	out, err := c.p.Recv()
	if out == nil {
		return nil, err
	}
	return []interface{}{out}, err
}

/*Terminate terminates the Client
closing the connection and the channels.*/
func (c *Client) Close() {
	c.p.conn.Close()
}

/*returns the last error that happened in the client goroutines*/
func (c *Client) Error() error {
	return c.p.Error()
}

func (c *Client) Start() {
	c.started = true
	go func() {
		defer c.Close()
		c.p.mouth()
	}()
	go func() {
		defer c.Close()
		c.p.ear()
	}()
}

func (c *Client) SetTimeout(t time.Duration) {
	c.p.rTimeout = t // racy c:
	c.p.wTimeout = t // racy c:
}
