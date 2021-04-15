package main

import (
	"bufio"
	"fmt"
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

func (n *command) Run(in <-chan string, out chan<- string) (err error) {
	cmd := exec.Command(n.cmd[0], n.cmd[1:]...)

	wg := sync.WaitGroup{}

	inPipe, _ := cmd.StdinPipe()
	wg.Add(1)
	go func() {
		for line := range in {
			inPipe.Write([]byte(line))
			inPipe.Write([]byte("\n"))
		}

		// let ran program know no more data is coming
		inPipe.Close()

		wg.Done()
	}()

	// Create a scanner which scans r in a line-by-line fashion
	outPipe, _ := cmd.StdoutPipe()
	scanner := bufio.NewScanner(outPipe)

	wg.Add(1)

	go func() {

		// Read line by line and process it
		for scanner.Scan() {
			line := scanner.Text()
			if line != "" {
				out <- line
			}
		}
		wg.Done()
	}()

	// Start the command and check for errors
	err = cmd.Run()
	wg.Wait()
	return err
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

		childIn := make(chan string, 100)
		childIns = append(childIns, childIn)

		go func(r runner, i int) {
			childErr := r.Run(childIn, out)
			if childErr != nil {
				err = multierror.Append(err, childErr)
			}
			wg.Done()
		}(child, i)
	}

	childrenInChan := make(chan string, 100)
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
