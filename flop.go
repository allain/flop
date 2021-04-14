package main

import (
	"fmt"
	"io"
	"log"
	"os/exec"
	"sync"
)

type logged struct {
	reader io.Reader
	writer io.Writer
}

func (lw *logged) Read(p []byte) (int, error) {
	n, err := lw.reader.Read(p)
	log.Printf("read %d %v", n, err)
	return n, err
}

func (lw *logged) Write(p []byte) (int, error) {
	n, err := lw.writer.Write(p)
	log.Printf("write %d %v", n, err)
	return n, err
}

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
		wg := sync.WaitGroup{}

		childWriters := make([]io.Writer, len(n.outs))
		childReaders := make([]*io.PipeReader, len(n.outs))

		for i, out := range n.outs {
			wg.Add(1)
			r, w := io.Pipe()
			childWriters[i] = w
			childReaders[i] = r

			go func(out runner, r *io.PipeReader, w *io.PipeWriter) {
				defer w.Close()
				out.Run(r, stdout, stderr)
				wg.Done()
			}(out, r, w)
		}

		w := io.MultiWriter(childWriters...)
		err := n.runner.Run(stdin, w, stderr)

		// let the child runners know there's not going to be any more data coming
		for _, cr := range childReaders {
			// childWriters[i].(*io.PipeWriter).Close()
			cr.Close()
		}

		wg.Wait()

		return err

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
