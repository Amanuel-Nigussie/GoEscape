package main

type Node struct {
	Value int
	Next  *int
}

var GlobalNode *Node

func RunStructSafe() {
	safeVal := 70
	n1 := &Node{Next: &safeVal}
	_ = n1
}

func RunStructLeaky() {
	leakVal := 80
	n2 := &Node{Next: &leakVal}
	GlobalNode = n2
}

func main() {
	dummy := 99
	localMainNode := Node{Value: 1, Next: &dummy}
	_ = localMainNode

	RunStructSafe()
	RunStructLeaky()
}