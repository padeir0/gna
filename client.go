package gna

import (
	"encoding/gob"
	"errors"
	"fmt"
	"net"
	"sync"
	"syscall"
	"time"
)

func Dial(addr string) (*Client, error) {
	c, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}
	cli := &Client{
		conn:    c,
		enc:     gob.NewEncoder(c),
		dec:     gob.NewDecoder(c),
		timeout: stdTimeout,
		mouthDt: make(chan []interface{}),
		acu:     &acumulator{dt: make([]*Input, 64)},
	}
	return cli, nil
}

type Client struct {
	conn    net.Conn
	enc     *gob.Encoder
	dec     *gob.Decoder
	timeout time.Duration
	mouthDt chan []interface{}
	wErr    error
	rErr    error

	acu     *acumulator
	started bool

	dead bool
	mu   sync.Mutex
}

func (c *Client) Send(dt ...interface{}) error {
	if c.started {
		c.mouthDt <- dt
		return nil
	}
	var err error
	for i := 0; i < len(dt) && err == nil; i++ {
		err = c.enc.Encode(&dt[i])
		if err != nil {
			err = fmt.Errorf("%w while encoding: %v", err, dt[i])
		}
	}
	return err
}

func (c *Client) Recv() ([]interface{}, error) {
	if c.started {
		dt := c.acu.consume()
		out := make([]interface{}, len(dt))
		for i := range dt {
			out[i] = dt[i].Data
		}
		return out, nil
	}
	var dt interface{}
	err := c.dec.Decode(&dt)
	return []interface{}{dt}, err
}

/*Terminate terminates the Client
closing the connection and the channels.*/
func (c *Client) Terminate() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.dead {
		c.dead = true
		c.conn.Close()
	}
}

/*returns the last error that happened in the client goroutines*/
func (c *Client) Error() error {
	if c.wErr != nil {
		if c.rErr != nil {
			return fmt.Errorf("%w, along with: %v", c.wErr, c.rErr)
		}
		return c.wErr
	}
	if c.rErr != nil {
		return c.rErr
	}
	return nil
}

func (c *Client) Start() {
	c.started = true
	go func() {
		defer c.Terminate()
		c.mouth()
	}()
	go func() {
		defer c.Terminate()
		c.ear()
	}()
}

func (c *Client) SetTimeout(t time.Duration) {
	c.timeout = t // racy
}

func (c *Client) mouth() {
	dt := []interface{}{}
	for {
		dt = <-c.mouthDt
		for i := range dt {
			err := c.enc.Encode(&dt[i])
			if err != nil {
				c.wErr = fmt.Errorf("%w while encoding %v", err, dt[i])
				if errors.Is(err, syscall.EPIPE) {
					return
				}
				break
			}

		}
	}
}

func (c *Client) ear() {
	for {
		err := c.conn.SetReadDeadline(time.Now().Add(c.timeout))
		if err != nil {
			c.rErr = fmt.Errorf("failed to set deadline: %w", err)
		}
		var dt interface{}
		err = c.dec.Decode(&dt)
		if err != nil {
			c.rErr = fmt.Errorf("recv: %w", err)
			if opErr, ok := err.(*net.OpError); ok && opErr.Timeout() {
				return
			}
			return
		}
		if dt != nil {
			c.acu.add(&Input{nil, dt})
		}
	}
}
