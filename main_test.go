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

type echo struct {
	runner
}

// copies input to output
func (er *echo) Run(stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
	io.Copy(stdout, stdin)
	return nil
}

type double struct {
	runner
}

// reads integers off stdin one at a time and sends their double to stdout
// if an error occurs, it's printed to stderr
func (dr *double) Run(stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
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
	n1 := newNode(&echo{})
	n2 := newNode(&echo{})

	result := n1.Pipe(n2)
	if result != n2 {
		t.Log("expected Pipe to return target node")
		t.Fail()
	}

	if len(n1.outs) != 1 {
		t.Log("expected node to be in outs")
		t.Fail()
	}
}

func TestCanWireUpNodesAsChain(t *testing.T) {
	n1 := newNode(&echo{})
	n2 := newNode(&echo{})
	n3 := newNode(&echo{})

	result := n1.Pipe(n2).Pipe(n3)
	if result != n3 {
		t.Fail()
	}
}

func TestErrorsWhenNodeAlreadyPiped(t *testing.T) {
	n1 := newNode(&echo{})
	n2 := newNode(&echo{})

	n1.Pipe(n2)

	defer func() {
		err := recover()
		if err == nil {
			t.Log("expected error on duplicate Pipe")
			t.Fail()
		}
	}()

	n1.Pipe(n2)

	t.Log("should panic when piping to same node twice")
	t.FailNow()
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
	c := double{}

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
	n := newNode(&echo{})

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
	n1 := newNode(&double{})
	n2 := newNode(&double{})
	n3 := newNode(&double{})

	n1.Pipe(n2).Pipe(n3)

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
	if outStr != "8\n16\n" {
		t.Logf("expected stdout to match 8\n16\n but was: %s", outStr)
		t.Fail()
	}

	errStr := stderr.String()
	if errStr != "" {
		t.Logf("expected no output on stderr: %s", outStr)
		t.Fail()
	}
}

func TestRunnerCanFanOut(t *testing.T) {
	n1 := newNode(&echo{})
	n2 := newNode(newCommand("awk", "{print toupper($0)}"))
	n3 := newNode(newCommand("awk", "{print tolower($0)}"))

	n1.Pipe(n2)
	n1.Pipe(n3)

	stdin := strings.NewReader("Hello\nWorld\n")
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
	lines := strings.Split(outStr, "\n")

	if len(lines) != 5 {
		t.Logf("expected 5 lines but received %d", len(lines))
		t.FailNow()
	}

	if outStr != "HELLO\nWORLD\nhello\nworld\n" {
		t.Logf("expected stdout to match HELLO\nWORLD\nhello\nworld\n but was: %s", outStr)
		t.FailNow()
	}

	errStr := stderr.String()
	if errStr != "" {
		t.Logf("expected no output on stderr: %s", outStr)
		t.FailNow()
	}
}
