package main

import (
	"fmt"
	"sort"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type echo struct{}

// copies input to output
func (er *echo) Run(in <-chan string, out chan<- string) error {
	for line := range in {
		out <- line
	}

	return nil
}

type double struct{}

// reads integers off stdin one at a time and sends their double to stdout
// if an error occurs, it's printed to stderr
func (dr *double) Run(in <-chan string, out chan<- string) error {
	for line := range in {
		number, err := strconv.ParseInt(line, 10, 64)
		if err != nil {
			return err
		}

		out <- fmt.Sprintf("%d", number*2)
	}

	return nil
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
		assert.Error(t, recover().(error))
	}()

	n1.Pipe(n2)

	assert.FailNow(t, "should panic when piping to same node twice")
}

func TestCommandRunnerWorks(t *testing.T) {
	c := command{cmd: []string{"echo", "testing"}}

	in := make(chan string, 1)
	in <- "This is a Test!"
	close(in)

	out := make(chan string, 100)

	err := c.Run(in, out)
	if err != nil {
		t.Log("unexpected error running command")
		t.FailNow()
	}

	close(out)

	assert.Equal(t, collectStrings(out), []string{"testing"})
}

func collectStrings(c chan string) []string {
	result := []string{}
	for line := range c {
		result = append(result, line)
	}
	return result
}

func TestDoubleRunnerWorks(t *testing.T) {
	c := double{}

	in := make(chan string, 100)
	in <- "1"
	in <- "2"
	in <- "3"
	close(in)

	out := make(chan string, 100)

	err := c.Run(in, out)
	assert.NoError(t, err)

	close(out)

	assert.Equal(t, []string{"2", "4", "6"}, collectStrings(out))
}

func TestNodeRunnerWorks(t *testing.T) {
	n := newNode(&echo{})

	in := make(chan string, 100)
	in <- "This is a test!"
	close(in)

	out := make(chan string, 100)

	err := n.Run(in, out)
	assert.NoError(t, err)

	close(out)

	assert.Equal(t, []string{"This is a test!"}, collectStrings(out))
}

func TestRunnerPipes(t *testing.T) {
	n1 := newNode(&double{})
	n2 := newNode(&double{})
	n3 := newNode(&double{})

	// n1.Pipe(n2)
	n1.Pipe(n2).Pipe(n3)

	in := make(chan string, 100)
	in <- "1"
	in <- "2"
	close(in)

	out := make(chan string, 100)

	err := n1.Run(in, out)
	assert.NoError(t, err)

	close(out)

	// assert.Equal(t, []string{"4", "8"}, collectStrings(out))
	assert.Equal(t, []string{"8", "16"}, collectStrings(out))
}

func skipTestWaitsForAllChildrenToFinish(t *testing.T) {
	n1 := newNode(&echo{})
	n2 := newNode(newCommand("sleep", "1"))
	n3 := newNode(newCommand("sleep", "1"))

	n1.Pipe(n2)
	n1.Pipe(n3)

	in := make(chan string, 100)
	in <- "1"
	in <- "2"
	close(in)

	out := make(chan string, 100)

	start := time.Now()
	err := n1.Run(in, out)
	assert.NoError(t, err)

	// close(out)
	// close(problems)
	end := time.Now()

	duration := end.Sub(start)
	if duration.Milliseconds() < 1000 {
		assert.FailNow(t, "should have ran for at least 1 second but ran for %dms", duration.Milliseconds())
	}

	if duration.Milliseconds() > 2000 {
		assert.FailNow(t, "should have run all nodes in parallel but seems they ran sequentially %dms", duration.Milliseconds())
	}
}

func TestRunnerCanFanOut(t *testing.T) {
	n1 := newNode(&echo{})
	n2 := newNode(newCommand("awk", "{print toupper($0)}"))
	n3 := newNode(newCommand("awk", "{print tolower($0)}"))

	n1.Pipe(n2)
	n1.Pipe(n3)

	in := make(chan string, 100)
	in <- "Hello"
	in <- "World"
	close(in)

	out := make(chan string, 100)

	err := n1.Run(in, out)
	assert.NoError(t, err)

	close(out)

	outLines := collectStrings(out)

	assert.Len(t, outLines, 4)

	s := sort.StringSlice(outLines)
	s.Sort()

	assert.Equal(t, []string{"HELLO", "WORLD", "hello", "world"}, []string(s))
}

type counter struct {
	n int
}

func (c *counter) Run(in <-chan string, out chan<- string) error {
	for i := 0; i < c.n; i++ {
		out <- strconv.Itoa(i)
	}

	return nil
}

func benchmarkLeaf(b *testing.B, runBuilder func() runner) {
	for i := 0; i < b.N; i++ {
		n1 := newNode(runBuilder())

		in := make(chan string, 100)
		close(in)

		out := make(chan string, 100)

		go func() {
			for range out {
			}
		}()

		err := n1.Run(in, out)
		assert.NoError(b, err)

		close(out)
	}
}

func BenchmarkLeaf(b *testing.B) {
	benchmarkLeaf(b, func() runner { return &counter{1000} })
}

func BenchmarkLeafCommand(b *testing.B) {
	benchmarkLeaf(b, func() runner { return newCommand("echo", "hello") })
}

func benchmarkChain(n int, b *testing.B, runBuilder func() runner) {
	for i := 0; i < b.N; i++ {
		root := newNode(&counter{1000})
		current := root
		for j := 0; j < n; j++ {
			current = current.Pipe(newNode(runBuilder()))
		}

		in := make(chan string, 100)
		close(in)

		out := make(chan string, 100)

		go func() {
			for range out {
			}
		}()

		err := root.Run(in, out)
		assert.NoError(b, err)

		close(out)
	}
}

func BenchmarkChain1(b *testing.B) {
	benchmarkChain(1, b, func() runner { return &echo{} })
}
func BenchmarkChainCommand1(b *testing.B) {
	benchmarkChain(1, b, func() runner { return newCommand("cat") })
}

func BenchmarkChain2(b *testing.B) {
	benchmarkChain(2, b, func() runner { return &echo{} })
}
func BenchmarkChainCommand2(b *testing.B) {
	benchmarkChain(2, b, func() runner { return newCommand("cat") })
}

func BenchmarkChain10(b *testing.B) {
	benchmarkChain(10, b, func() runner { return &echo{} })
}

func BenchmarkChainCommand10(b *testing.B) {
	benchmarkChain(10, b, func() runner { return newCommand("cat") })
}

func benchmarkFan(n int, b *testing.B, runBuilder func() runner) {
	for i := 0; i < b.N; i++ {
		n1 := newNode(&counter{1000})
		for j := 0; j < n; j++ {
			n1.Pipe(newNode(runBuilder()))
		}

		in := make(chan string, 100)
		close(in)

		out := make(chan string, 100)

		go func() {
			for range out {
			}
		}()

		err := n1.Run(in, out)
		assert.NoError(b, err)

		close(out)

	}
}

func BenchmarkFan2(b *testing.B) {
	benchmarkFan(2, b, func() runner { return &echo{} })
}
func BenchmarkFanCommand2(b *testing.B) {
	benchmarkFan(2, b, func() runner { return newCommand("cat") })
}

func BenchmarkFan10(b *testing.B) {
	benchmarkFan(10, b, func() runner { return &echo{} })
}

func BenchmarkFanCommand10(b *testing.B) {
	benchmarkFan(10, b, func() runner { return newCommand("cat") })
}
