package main

var GlobalFunc func()

func RunClosureSafe() {
	safeVal := 110
	f := func() {
		safeVal++
	}
	f()
}

func RunClosureLeaky() {
	leakVal := 120
	GlobalFunc = func() {
		leakVal++
	}
}

func main() {
	defer RunClosureSafe()
	execute := func() {
		RunClosureLeaky()
	}
	execute()
}