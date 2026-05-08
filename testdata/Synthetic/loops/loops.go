package main

var GlobalLoopSink *int

func RunLoopSafe() {
	// SAFE: Pointer created and dies inside the loop
	for i := 0; i < 5; i++ {
		safeVal := i
		localPtr := &safeVal
		_ = localPtr
	}
}

func RunLoopLeaky() {
	// LEAKY: Pointer escapes the loop to global
	for j := 0; j < 5; j++ {
		leakVal := j
		if j == 3 {
			GlobalLoopSink = &leakVal
		}
	}
}

func main() {
	for iter := 0; iter < 2; iter++ {
		if iter%2 == 0 {
			RunLoopSafe()
		}
		RunLoopLeaky()
	}
}
