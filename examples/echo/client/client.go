package main

import (
	"fmt"
	"github.com/kazhmir/gna"
	"time"
)

func main() {
	c, err := Connect()
	if err != nil {
		fmt.Println(err)
		return
	}
	err = Loop(c)
	if err != nil {
		fmt.Println(err)
	}
}

func Loop(c *gna.Client) error {
	for {
		c.Dispatch("beep")
		if err := c.Error(); err != nil {
			return err
		}
		fmt.Println(c.RecvBatch())
		time.Sleep(30 * time.Millisecond)
	}
}

func Connect() (*gna.Client, error) {
	c, err := gna.Dial(":8888")
	if err != nil {
		return nil, err
	}
	err = c.Send("banana")
	if err != nil {
		return nil, err
	}
	c.Start()
	return c, nil
}
