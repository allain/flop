package main

import (
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"sync"

	multierror "github.com/hashicorp/go-multierror"
)

type runner interface {
	Run(in <-chan string, out chan<- string) error
}

func newCommand(cmd string, args ...string) *command {
	c := command{cmd: append([]string{cmd}, args...)}
	return &c
}

type command struct {
	cmd []string
}

type chanReader struct {
	c   <-chan string
	buf bytes.Buffer
}

func (c *chanReader) Read(p []byte) (int, error) {
	line, ok := <-c.c
	if !ok {
		return 0, io.EOF
	}

	c.buf.WriteString(line)
	c.buf.WriteRune('\n')

	return c.buf.Read(p)
}

type chanWriter struct {
	c   chan<- string
	buf bytes.Buffer
}

func (cw *chanWriter) Write(p []byte) (int, error) {
	n, err := cw.buf.Write(p)
	if err != nil {
		return 0, err
	}

	for {
		line, err := cw.buf.ReadString('\n')
		if err != nil {
			break
		}
		cw.c <- line[:len(line)-1]
	}

	return n, err
}

func (n *command) Run(in <-chan string, out chan<- string) (err error) {
	cmd := exec.Command(n.cmd[0], n.cmd[1:]...)

	inReader := chanReader{c: in}
	cmd.Stdin = &inReader

	outWriter := chanWriter{c: out}
	cmd.Stdout = &outWriter

	return cmd.Run()
}

func newNode(r runner) *node {
	n := node{runner: r}
	return &n
}

type node struct {
	runner runner
	outs   []*node
}

func (n *node) Run(in <-chan string, out chan<- string) (err error) {
	if len(n.outs) == 0 {
		return n.runner.Run(in, out)
	}

	wg := sync.WaitGroup{}

	childIns := []chan string{}

	for i, child := range n.outs {
		wg.Add(1)

		childIn := make(chan string)
		childIns = append(childIns, childIn)

		go func(r runner, i int) {
			childErr := r.Run(childIn, out)
			if childErr != nil {
				err = multierror.Append(err, childErr)
			}
			wg.Done()
		}(child, i)
	}

	childrenInChan := make(chan string)
	go func() {
		for line := range childrenInChan {
			for _, childIn := range childIns {
				childIn <- line
			}
		}
		for _, childIn := range childIns {
			close(childIn)
		}
	}()

	rootErr := n.runner.Run(in, childrenInChan)
	if rootErr != nil {
		err = multierror.Append(err, rootErr)
	}

	close(childrenInChan)

	wg.Wait()

	return err
}

func (n *node) Pipe(n2 *node) *node {
	index := n.findNode(n2)
	if index != -1 {
		panic(fmt.Errorf("duplicate piping"))
	}

	n.outs = append(n.outs, n2)
	return n2
}

func (n *node) findNode(n2 *node) int {
	for i, out := range n.outs {
		if out == n2 {
			return i
		}
	}
	return -1
}
