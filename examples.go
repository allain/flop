package main

import (
	"fmt"
	"strconv"
	"time"
)

type counter2 struct {
	n     int
	delay time.Duration
}

func (c *counter2) Run(in <-chan string, out chan<- string) error {
	for i := 0; i < c.n; i++ {
		out <- strconv.Itoa(i)
		time.Sleep(c.delay)
	}

	return nil
}

func runNode(n *node) error {
	in := make(chan string)
	close(in)

	out := make(chan string)
	go func() {
		for line := range out {
			fmt.Println(line)
		}
	}()

	err := n.Run(in, out)

	close(out)

	return err
}

func main() {
	fmt.Println("Piping of interactive program")
	countRunner := counter2{n: 3, delay: time.Millisecond * 1000}
	catRunner := newCommand("cat")
	example1 := newNode(&countRunner)
	example1.Pipe(newNode(catRunner))
	runNode(example1)

	fmt.Println("Example 5 - Fanout")
	pingRunner := newCommand("ping", "-c", "3", "www.google.com")
	upperRunner := newCommand("awk", "{print toupper($0); fflush()}")
	lowerRunner := newCommand("awk", "{print tolower($0); fflush()}")
	example2 := newNode(pingRunner)
	example2.Pipe(newNode(upperRunner))
	example2.Pipe(newNode(lowerRunner))
	runNode(example2)

}
