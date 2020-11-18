package mgs

import (
	"encoding/binary"
)

var (
	/*ErrServerFull happens when len(Server.talkers) == Server.MaxPlayers*/
	ErrServerFull = &ServerError{0, "server is full"}
	/*ErrBadPacketSize happens when the client sends the wrong Header*/
	ErrBadPacketSize = &ServerError{1, "packet size sent by client is of wrong lenght"}
)

/*ServerError is used to comunicate an error from the server to the client
and also for logging.
*/
type ServerError struct {
	Code    uint16
	Message string
}

func (e *ServerError) Error() string {
	return e.Message
}

/*Type of ServerErrors are aways 0*/
func (e *ServerError) Type() byte {
	return 0
}

/*Size is aways 2 (the error code) + the size of the message*/
func (e *ServerError) Size() int {
	return 2 + len(e.Message)
}

/*Encode encodes the code and message respectivelly*/
func (e *ServerError) Encode(buff []byte) error {
	binary.BigEndian.PutUint16(buff, e.Code)
	for i := range e.Message {
		buff[2+i] = e.Message[i]
	}
	return nil
}
