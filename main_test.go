package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strconv"
	"strings"
	"testing"
)

type echoRunner struct {
	runner
}

// copies input to output
func (er *echoRunner) Run(stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
	io.Copy(stdout, stdin)
	return nil
}

type doubleRunner struct {
	runner
}

// reads integers off stdin one at a time and sends their double to stdout
// if an error occurs, it's printed to stderr
func (dr *doubleRunner) Run(stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
	r := bufio.NewReader(stdin)
	w := bufio.NewWriter(stdout)
	e := bufio.NewWriter(stderr)

	var resultErr error

	for {
		lineBytes, _, err := r.ReadLine()
		line := string(lineBytes)

		if err != nil {
			if err != io.EOF {
				fmt.Fprintf(e, "invalid integer: %s\n", line)
				resultErr = err
			}
			break
		}

		number, err := strconv.ParseInt(line, 10, 64)
		if err != nil {
			fmt.Fprintf(e, "invalid integer: %s\n", line)
			resultErr = err
			break
		}

		fmt.Fprintf(w, "%d\n", number*2)
	}

	w.Flush()
	e.Flush()

	return resultErr
}

func TestCanWireUpNodes(t *testing.T) {
	n1 := newNode(&echoRunner{})
	n2 := newNode(&echoRunner{})

	err := n1.Pipe(n2)
	if err != nil {
		t.Fail()
	}

	if len(n1.outs) != 1 {
		t.Log("expected node to be in outs")
		t.Fail()
	}
}

func TestCanUnwireNodes(t *testing.T) {
	n1 := newNode(&echoRunner{})
	n2 := newNode(&echoRunner{})

	n1.Pipe(n2)
	n1.Unpipe(n2)

	if len(n1.outs) != 0 {
		t.Log("expected node not to be in outs")
		t.Fail()
	}
}

func TestErrorsWhenNodeAlreadyPiped(t *testing.T) {
	n1 := newNode(&echoRunner{})
	n2 := newNode(&echoRunner{})

	n1.Pipe(n2)
	err := n1.Pipe(n2)

	if err == nil {
		t.Log("expected error on duplicate Pipe")
		t.Fail()
	}
}

func TestErrorsWhenRemovingMissingNode(t *testing.T) {
	n1 := newNode(&echoRunner{})
	n2 := newNode(&echoRunner{})

	err := n1.Unpipe(n2)

	if err == nil {
		t.Log("unpiping missing node should error")
		t.Fail()
	}
}

func TestCommandRunnerWorks(t *testing.T) {
	c := command{cmd: []string{"echo", "testing"}}

	stdin := strings.NewReader("This is a test!")
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	err := c.Run(stdin, stdout, stderr)
	if err != nil {
		t.Log("Error while running unkillable program")
		t.Fail()
	}

	if stdin.Len() != 0 {
		t.Log("expected stdin to be read completely")
		t.Fail()
	}

	outStr := stdout.String()
	if outStr != "testing\n" {
		t.Logf("expected stdout to match testing but was: %s", outStr)
		t.Fail()
	}

	errStr := stderr.String()
	if errStr != "" {
		t.Logf("expected no output on stderr: %s", outStr)
		t.Fail()
	}
}

func TestDoubleRunnerWorks(t *testing.T) {
	c := doubleRunner{}

	stdin := strings.NewReader("1\n2\n3")
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	err := c.Run(stdin, stdout, stderr)
	if err != nil {
		t.Log("Error while running doubler")
		t.Fail()
	}

	if stdin.Len() != 0 {
		t.Log("expected stdin to be read completely")
		t.Fail()
	}

	outStr := stdout.String()
	if outStr != "2\n4\n6\n" {
		t.Logf("expected stdout to match 2\n4\n6 but was: %s", outStr)
		t.Fail()
	}

	errStr := stderr.String()
	if errStr != "" {
		t.Logf("expected no output on stderr: %s", outStr)
		t.Fail()
	}
}

func TestNodeRunnerWorks(t *testing.T) {
	n := newNode(&echoRunner{})

	stdin := strings.NewReader("This is a test!")
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	err := n.Run(stdin, stdout, stderr)
	if err != nil {
		t.Log("Error while running program")
		t.Fail()
	}

	if stdin.Len() != 0 {
		t.Log("expected stdin to be read completely")
		t.Fail()
	}

	outStr := stdout.String()
	if outStr != "This is a test!" {
		t.Logf("expected stdout to match stdin but was: %s", outStr)
		t.Fail()
	}

	errStr := stderr.String()
	if errStr != "" {
		t.Logf("expected no output on stderr: %s", outStr)
		t.Fail()
	}

}

func TestRunnerPipes(t *testing.T) {
	n1 := newNode(&doubleRunner{})
	n2 := newNode(&doubleRunner{})

	n1.Pipe(n2)

	stdin := strings.NewReader("1\n2\n")
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	err := n1.Run(stdin, stdout, stderr)
	if err != nil {
		t.Log("Error while running program")
		t.Fail()
	}

	if stdin.Len() != 0 {
		t.Log("expected stdin to be read completely")
		t.Fail()
	}

	outStr := stdout.String()
	if outStr != "4\n8\n" {
		t.Logf("expected stdout to match 4\n8\n but was: %s", outStr)
		t.Fail()
	}

	errStr := stderr.String()
	if errStr != "" {
		t.Logf("expected no output on stderr: %s", outStr)
		t.Fail()
	}
}
