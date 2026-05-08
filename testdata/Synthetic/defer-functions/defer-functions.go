package main

var GlobalDeferSink *int

func dummyDeferSafe(v int) {}
func dummyDeferLeak(p *int) {
	GlobalDeferSink = p
}

func RunDeferTest() {
	safeVal := 230
	defer dummyDeferSafe(safeVal)

	leakVal := 240
	defer dummyDeferLeak(&leakVal)
}

func main() {
	defer func() {
		defer RunDeferTest()
	}()
}
