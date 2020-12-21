package gna

import (
	"encoding/gob"
	"errors"
	"fmt"
	"net"
	"time"
)

type shipper interface {
	/*sends the data to the right chan for dispatching*/
	ship(interface{})
}

/*dispatcher runs above a persistent TCP connection with gob encoding,
it acts as writer and owner of the connection.
*/
type dispatcher struct {
	conn net.Conn
	enc  *gob.Encoder
	dec  *gob.Decoder
	err  error // encode/write error

	cDisp chan interface{}

	rTimeout    time.Duration
	wTimeout    time.Duration
	shouldStart bool // only used after auth, not concurrently
}

func (p *dispatcher) ship(dt interface{}) {
	select {
	case p.cDisp <- dt:
	default:
		/* this means
		a: There is a faulty or intentionally bad receiver
		b: The server resources are being overwelmed
		in both cases closing the connection and clearing resources is needed.
		*/
		p.err = errors.New("full buffer")
		p.Close()
	}
}

/*Send sets the deadline and encodes the data, it may halt, but it returns the error
in case of failure, guaranteeing knowledge if the user has the data.
This differs from pConn.ship() in which it's only known after the connection is closed.*/
func (p *dispatcher) Send(dt interface{}) error {
	err := p.conn.SetWriteDeadline(time.Now().Add(15 * time.Second))
	if err != nil {
		return err
	}
	return p.enc.Encode(&dt)
}

/*Recv sets the deadline and decodes data from the connection,
you cannot use this safely after the receiver is started*/
func (p *dispatcher) Recv() (interface{}, error) {
	err := p.conn.SetReadDeadline(time.Now().Add(p.rTimeout))
	if err != nil {
		err = fmt.Errorf("failed to set deadline: %w", err)
		return nil, err
	}
	var dt interface{}
	err = p.dec.Decode(&dt)
	return dt, err
}

/*Error returns the error that caused the pConn to disconnect.*/
func (p *dispatcher) Error() error {
	return p.err
}

/*Close terminates the player, closing the connection.*/
func (p *dispatcher) Close() error {
	p.shouldStart = false // used in auth
	return p.conn.Close()
}

func (p *dispatcher) work() {
	defer p.Close()
	for {
		dt := <-p.cDisp
		err := p.Send(dt)
		if err != nil {
			if p.err == nil {
				p.err = err
			}
			return
		}
	}
}
