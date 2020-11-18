package mgs

import (
	"encoding/binary"
)

var (
	ErrServerFull    = &ServerError{0, "server is full"}
	ErrBadPacketSize = &ServerError{1, "packet size sent by client is of wrong lenght"}
)

type ServerError struct {
	Code    uint16
	Message string
}

func (e *ServerError) Error() string {
	return e.Message
}

func (e *ServerError) Type() byte {
	return 0
}

func (e *ServerError) Size() int {
	return 2
}

func (e *ServerError) Encode(buff []byte) error {
	binary.BigEndian.PutUint16(buff, e.Code)
	return nil
}
