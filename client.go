package gna

import (
	"encoding/gob"
	"fmt"
	"net"
	"sync"
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
		acu: &cliBucket{dt: make([]interface{}, 64)},
		dispatcher: dispatcher{
			conn:        c,
			enc:         gob.NewEncoder(c),
			dec:         gob.NewDecoder(c),
			rTimeout:    stdReadTimeout,
			wTimeout:    stdWriteTimeout,
			cDisp:       make(chan interface{}),
			shouldStart: true,
		},
	}
	return cli, nil
}

/*Client abstracts the connection handling and communication with the server.*/
type Client struct {
	acu     *cliBucket
	err     error
	started bool
	dispatcher
}

/*Send sets the deadline and encodes the data*/
func (c *Client) Send(dt interface{}) error {
	err := c.dispatcher.Send(dt)
	if err != nil {
		err = fmt.Errorf("%w while encoding: %v", err, dt)
	}
	return err
}

/*Dispatch is like send but it doesn't halt and doesn't guarantee delivery.
If used with a unstarted Client it panics.*/
func (c *Client) Dispatch(data interface{}) {
	if c.started {
		c.cDisp <- data
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
	out, err := c.dispatcher.Recv()
	if out == nil {
		return nil, err
	}
	return out, err
}

/*RecvBatch empties the acumulator, retrieving the data*/
func (c *Client) RecvBatch() []interface{} {
	if c.started {
		return c.acu.consume()
	}
	return nil
}

/*Start starts the client receiver and dispatcher*/
func (c *Client) Start() {
	c.started = true
	go c.dispatcher.work()
	go c.receiver()
}

/*SetTimeout sets both read and write timeout*/
func (c *Client) SetTimeout(t time.Duration) {
	c.rTimeout = t // racy c:
	c.wTimeout = t // racy c:
}

func (c *Client) Error() error {
	if c.err != nil {
		if c.dispatcher.err != nil {
			return fmt.Errorf("%w, alongside: %v", c.err, c.dispatcher.err)
		}
		return c.err
	}
	if c.dispatcher.err != nil {
		return c.dispatcher.err
	}
	return nil
}

func (c *Client) receiver() {
	defer c.Close()
	for {
		dt, err := c.dispatcher.Recv()
		if err != nil {
			c.err = fmt.Errorf("recv: %w", err)
			if opErr, ok := err.(*net.OpError); ok && opErr.Timeout() {
				return
			}
			return
		}
		if dt != nil { // ?
			c.acu.add(dt)
		}
	}
}

type cliBucket struct {
	dt []interface{}
	i  int
	mu sync.Mutex
}

func (is *cliBucket) add(dt interface{}) {
	is.mu.Lock()
	if is.i >= len(is.dt) {
		is.dt = append(is.dt, make([]interface{}, 64)...)
	}
	is.dt[is.i] = dt
	is.i++
	is.mu.Unlock()
}

func (is *cliBucket) consume() []interface{} {
	is.mu.Lock()
	out := make([]interface{}, is.i)
	copy(out, is.dt[:is.i])
	is.i = 0
	is.mu.Unlock()
	return out
}
