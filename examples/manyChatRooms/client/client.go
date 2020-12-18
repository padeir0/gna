package client

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/kazhmir/gna"
	"github.com/kazhmir/gna/examples/manyChatRooms/shared"
)

func connect(addr, name string) (*gna.Client, error) {
	cli, err := gna.Dial(addr)
	if err != nil {
		return nil, err
	}
	err = cli.Send(shared.CliAuth{Name: name})
	if err != nil {
		return nil, err
	}
	dts, err := cli.Recv()
	if v, ok := dts.(shared.SrAuth); ok {
		fmt.Printf("UserID: %v\n", v.UserID)
	}
	cli.Start()
	return cli, err
}

func NewC(addr, name string, term *bufio.Reader) (*C, error) {
	cli, err := connect(addr, name)
	if err != nil {
		return nil, err
	}
	cli.SetTimeout(60 * time.Second)
	return &C{
		cli:  cli,
		term: term,
	}, nil
}

type C struct {
	cli  *gna.Client
	term *bufio.Reader
}

func (c *C) Start() {
	go c.update()
	c.processInputs()
}

func (c *C) update() {
	ticker := time.NewTicker(50 * time.Millisecond)
	for {
		<-ticker.C
		data := c.cli.RecvBatch()
		if err := c.cli.Error(); err != nil {
			log.Fatalf("Recv Error: %v\n", err)
		}
		if len(data) == 0 {
			continue
		}
		for _, dt := range data {
			switch v := dt.(type) {
			case shared.Message:
				fmt.Printf("%v: %v\n", v.Username, v.Data)
			default:
				fmt.Printf("\n%#v, %T\n", dt, dt)
			}
		}
	}
}

func (c *C) processInputs() {
	for {
		msg, err := c.term.ReadString('\n')
		if err != nil {
			fmt.Println(err)
			continue
		}
		dt := parse(strings.Replace(msg, "\n", "", -1))
		if dt != nil {
			c.cli.Dispatch(dt)
			if err := c.cli.Error(); err != nil {
				log.Fatalf("Send Error: %v\n", err)
			}
		}
	}
}

/*Commands follow the basic syntax: /<command> <data>
where: <command> == quit|room|chname
and <data> == [^\n]*
parse returns the data and command based on the list above
*/
func parse(line string) interface{} {
	r, size := utf8.DecodeRuneInString(line)
	if r == utf8.RuneError {
		if size == 0 {
			return nil
		}
		log.Fatal("invalid rune")
	}
	if r == '/' {
		out := parseCmd(line[size:])
		if out != nil {
			return out
		}
		return nil
	}
	return line // string is considered a message
}

func parseCmd(line string) *shared.Cmd {
	s := strings.SplitN(line, " ", 2)
	if len(s) < 2 {
		t := noDataCmd(s[0])
		if t == shared.Error {
			return nil
		}
		return &shared.Cmd{T: t}
	}
	t := cmdWithData(s[0])
	if t == shared.Error {
		return nil
	}
	return &shared.Cmd{Data: s[1], T: t}
}

func noDataCmd(cmd string) int {
	switch cmd {
	case "quit":
		os.Exit(0)
	case "num":
		return shared.Num
	}
	return shared.Error
}

func cmdWithData(cmd string) int {
	switch cmd {
	case "room":
		return shared.CRoom
	case "rename":
		return shared.CName
	}
	return shared.Error
}
