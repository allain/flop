package main

import (
	"log"
)

func runNode(n *node) error {
	in := make(chan string)
	close(in)

	out := make(chan string)
	go func() {
		for line := range out {
			log.Println(line, out)
		}
	}()

	err := n.Run(in, out)

	close(out)

	return err
}

func example1() {
	log.Println("Example 1")
	n1 := newNode(newCommand("date"))

	runNode(n1)
}

func example2() {
	log.Println("Example 2")
	n1 := newNode(newCommand("date"))
	n2 := newNode(newCommand("awk", "{print tolower($0)}"))
	n1.Pipe(n2)

	runNode(n1)
}

func example3() {
	log.Println("Example 3")
	n1 := newNode(newCommand("ping", "-c", "3", "www.google.com"))
	runNode(n1)
}

func example4() {
	log.Println("Example 4")
	n1 := newNode(newCommand("ping", "-c", "3", "www.google.com"))
	n2 := newNode(newCommand("awk", "{print toupper($0)}"))

	n1.Pipe(n2)
	runNode(n1)
}

func example5() {
	log.Println("Example 5")
	n1 := newNode(newCommand("ping", "-c", "10", "www.google.com"))
	n2 := newNode(newCommand("awk", "{print toupper($0)}"))
	n3 := newNode(newCommand("awk", "{print tolower($0)}"))

	n1.Pipe(n2)
	n1.Pipe(n3)

	runNode(n1)
}

func main() {
	// example1()
	// example2()
	// example3()
	example4()
	// example5()
}
