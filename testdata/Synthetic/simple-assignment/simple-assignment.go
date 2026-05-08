package main

var GlobalAssignSink *int

func RunAssignmentSafe() {
	// SAFE: Stays on STACK
	safeVal := 10
	localPtr := &safeVal
	_ = localPtr
}

func RunAssignmentLeaky() {
	// LEAKY: Escapes to global HEAP
	leakVal := 20
	leakPtr := &leakVal
	GlobalAssignSink = leakPtr
}

func main() {
	executeLeaky := true
	if executeLeaky {
		RunAssignmentLeaky()
	} else {
		RunAssignmentSafe()
	}
	RunAssignmentSafe()
}
