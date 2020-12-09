package mgs

import (
	//	"bufio"
	"encoding/gob"
	"errors"
	//	"fmt"
	"log"
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
	//	r := bufio.NewReader(c)
	//	w := bufio.NewWriter(c)
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

	acu *acumulator

	dead bool
	mu   sync.Mutex
}

func (c *Client) Send(dt interface{}) error {
	return c.enc.Encode(&dt)
}
func (c *Client) Recv() (interface{}, error) {
	var dt interface{}
	err := c.dec.Decode(&dt)
	return dt, err
}

func (c *Client) RecvAll() []interface{} {
	dt := c.acu.consume()
	out := make([]interface{}, len(dt))
	for i := range dt {
		out[i] = dt[i].Data
	}
	return out
}

/*Terminate terminates the Client
closing the connection and the channels.*/
func (c *Client) Terminate() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.dead {
		c.dead = true
		close(c.mouthDt)
	}
}

func (c *Client) Start() {
	go func() {
		defer c.Terminate()
		c.mouth()
	}()
	go func() {
		defer c.Terminate()
		c.ear()
	}()
}

func (c *Client) mouth() {
	dt := []interface{}{}
	for {
		dt = <-c.mouthDt
		for i := range dt {
			err := c.enc.Encode(dt[i])
			if errors.Is(err, syscall.EPIPE) { // ?
				log.Println("Cancelling cackets.", err)
				return
			}
			if err != nil {
				log.Println(err)
				break
			}

		}
	}
}

func (c *Client) ear() {
	for {
		err := c.conn.SetReadDeadline(time.Now().Add(c.timeout))
		if err != nil {
			log.Println(err)
		}
		var dt interface{}
		err = c.dec.Decode(&dt)
		if err != nil {
			log.Println(err)
			/*if opErr, ok := err.(*net.OpError); ok && opErr.Timeout() {
				return
			}*/
			return
		}
		if dt != nil { // ?
			c.acu.add(&Input{nil, dt})
		}
	}
}
