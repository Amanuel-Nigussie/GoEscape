package main

var GlobalMultiSink *int

func getSafeTuple() (int, int) {
	return 1, 2
}

func getLeakyTuple(p *int) (int, *int) {
	return 1, p
}

func RunMultiReturnTest() {
	safeVal1, safeVal2 := getSafeTuple()
	_ = safeVal1
	_ = safeVal2

	leakVal := 310
	_, leakedPtr := getLeakyTuple(&leakVal)
	GlobalMultiSink = leakedPtr
}

func main() {
	_, _ = getSafeTuple()
	RunMultiReturnTest()
}
