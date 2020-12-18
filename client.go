package gna

import (
	"encoding/gob"
	"fmt"
	"net"
	"time"
)

/*Dial tries to connect to the address. If any error is encountered it returns
a nil *Client and a non-nil error.*/
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

/*Client abstracts the connection handling and communication with the server.*/
type Client struct {
	p       *Player
	started bool
}

/*Send sets the deadline and encodes the data*/
func (c *Client) Send(dt interface{}) error {
	err := c.p.Send(dt)
	if err != nil {
		err = fmt.Errorf("%w while encoding: %v", err, dt)
	}
	return err
}

/*Dispatch is like send but it doesn't halt and doesn't guarantee delivery.
If used with a unstarted Client it panics.*/
func (c *Client) Dispatch(data interface{}) {
	if c.started {
		c.p.cDisp <- data
		return
	}
	panic("cannot dispatch, client not started")
}

/*Recv sets the deadline and encodes the data, if used after the Client
has started it panics.*/
func (c *Client) Recv() (interface{}, error) {
	if c.started {
		panic("recv cannot be used safely after Client has started")
	}
	out, err := c.p.Recv()
	if out == nil {
		return nil, err
	}
	return out, err
}

/*RecvBatch empties the acumulator, retrieving the data*/
func (c *Client) RecvBatch() []interface{} {
	if c.started {
		dt := c.p.acu.consume()
		out := make([]interface{}, len(dt))
		for i := range dt {
			out[i] = dt[i].Data
		}
		return out
	}
	return nil
}

/*Close closes the Client connection*/
func (c *Client) Close() error {
	return c.p.conn.Close()
}

/*Error returns the last error that happened in the client goroutines*/
func (c *Client) Error() error {
	return c.p.Error()
}

/*Start starts the client receiver and dispatcher*/
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

/*SetTimeout sets both read and write timeout*/
func (c *Client) SetTimeout(t time.Duration) {
	c.p.rTimeout = t // racy c:
	c.p.wTimeout = t // racy c:
}
