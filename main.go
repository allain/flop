package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"
)

type runner interface {
	Run(stdin io.Reader, stdout io.Writer, stderr io.Writer) error
}

func newCommand(cmd string, args ...string) *command {
	c := command{cmd: append([]string{cmd}, args...)}

	return &c
}

type command struct {
	cmd []string
}

func (n *command) Run(stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
	cmd := exec.Command(n.cmd[0], n.cmd[1:]...)
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr

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

func (n *node) Run(stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
	if len(n.outs) == 0 {
		// simple case where it's a leaf node, just do the work
		return n.runner.Run(stdin, stdout, stderr)
	}

	if len(n.outs) > 1 {
		panic("TODO: handle more than one piped runner")
	}

	r, w := io.Pipe()
	defer w.Close()

	wg := sync.WaitGroup{}

	for _, out := range n.outs {
		wg.Add(1)
		go func(out runner) {
			out.Run(r, stdout, stderr)
			wg.Done()
		}(out)
	}

	err := n.runner.Run(stdin, w, stderr)

	// let the runner know there's not going to be any more data coming
	w.Close()

	wg.Wait()

	return err
}

func (n *node) Pipe(n2 *node) error {
	index := n.findNode(n2)
	if index != -1 {
		return fmt.Errorf("duplicate piping")
	}

	n.outs = append(n.outs, n2)
	return nil
}

func (n *node) Unpipe(n2 *node) error {
	i := 0
	found := false
	for _, out := range n.outs {
		if out == n2 {
			found = true
		} else {
			n.outs[i] = out
			i++
		}
	}

	if !found {
		return fmt.Errorf("node not found")
	}

	// erasing truncated values
	for j := i; j < len(n.outs); j++ {
		n.outs[j] = nil
	}
	n.outs = n.outs[:i]
	return nil
}

func (n *node) findNode(n2 *node) int {
	for i, out := range n.outs {
		if out == n2 {
			return i
		}
	}
	return -1
}

func example1() {
	log.Println("Example 1")
	n1 := newNode(newCommand("date"))
	n1.Run(strings.NewReader(""), os.Stdout, os.Stderr)
}

func example2() {
	log.Println("Example 2")
	n1 := newNode(newCommand("date"))
	n2 := newNode(newCommand("awk", "{print tolower($0)}"))
	n1.Pipe(n2)

	n1.Run(strings.NewReader(""), os.Stdout, os.Stderr)
}

func example3() {
	log.Println("Example 3")
	n1 := newNode(newCommand("ping", "-c", "3", "www.google.com"))
	n1.Run(strings.NewReader(""), os.Stdout, os.Stderr)
}

func example4() {
	log.Println("Example 4")
	n1 := newNode(newCommand("ping", "-c", "3", "www.google.com"))
	n2 := newNode(newCommand("awk", "{print toupper($0)}"))

	n1.Pipe(n2)
	n1.Run(strings.NewReader(""), os.Stdout, os.Stderr)
}

func main() {
	example1()
	example2()
	example3()
	example4()
}
