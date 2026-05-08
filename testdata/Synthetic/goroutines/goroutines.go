package main

func dummyVal(v int) {}
func dummyPtr(p *int) {}

func RunGoroutineSafe() {
	safeVal := 150
	go dummyVal(safeVal)
}

func RunGoroutineLeaky() {
	leakVal := 160
	go dummyPtr(&leakVal)
}

func main() {
	go RunGoroutineSafe()
	go func() {
		RunGoroutineLeaky()
	}()
}