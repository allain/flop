package main

import (
	"log"
	"os"
	"strings"
)

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
	n1 := newNode(newCommand("ping", "www.google.com"))
	n2 := newNode(newCommand("awk", "{print toupper($0)}"))

	n1.Pipe(n2)
	n1.Run(strings.NewReader(""), os.Stdout, os.Stderr)
}

func example5() {
	log.Println("Example 5")
	n1 := newNode(newCommand("ping", "-c", "3", "www.google.com"))
	n2 := newNode(newCommand("awk", "{print toupper($0)}"))
	n3 := newNode(newCommand("awk", "{print tolower($0)}"))

	n1.Pipe(n2)
	n1.Pipe(n3)
	n1.Run(strings.NewReader(""), os.Stdout, os.Stderr)
}

func main() {
	example1()
	example2()
	example3()
	example4()
	example5()
}
