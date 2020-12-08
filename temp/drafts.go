package main

import (
	"fmt"
	"os"
	"os/signal"
	"sync"
	"time"
)

const size = 5

func main() {
	wl := make([]*worker, size)
	sigint := make(chan os.Signal, 1)
	signal.Notify(sigint, os.Interrupt)
	for i := 0; i < size; i++ {
		wl[i] = &worker{
			sig:  make(chan *sync.WaitGroup),
			data: make(chan int),
		}
		go wl[i].work()
	}
Loop:
	for {
		select {
		case <-sigint:
			for i := range wl {
				wl[i].kill()
			}
			break Loop
		default:
			start := time.Now()
			go func() {
				defer func() {
					if r := recover(); r != nil {
						fmt.Println(r)
					}
				}()
				var wg sync.WaitGroup
				wg.Add(1)
				for i := range wl {
					wl[i].sig <- &wg
					wl[i].data <- i
				}
				fmt.Println("signaling start")
				wg.Done()
			}()
			end := time.Now()
			fmt.Println(end.Sub(start))
			time.Sleep(500 * time.Millisecond)
		}
	}
	time.Sleep(1200 * time.Millisecond)
}

func task(n int) int {
	out := 1
	for i := 1; i < n; i++ {
		out *= i
	}
	return out
}

type worker struct {
	id   int
	sig  chan *sync.WaitGroup
	data chan int
}

func (w *worker) work() {
	for {
		sig := <-w.sig
		if sig == nil {
			return
		}
		amount := <-w.data
		sig.Wait()
		time.Sleep(time.Millisecond * 30) // waiting for very long syscall
		fmt.Println(task(amount))
	}
}

func (w *worker) kill() {
	close(w.sig)
}
