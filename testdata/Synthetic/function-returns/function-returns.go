package main

var MainReturnSink *int

func returnSafeValue() int {
	safeVal := 30
	return safeVal
}

func returnLeakyPointer() *int {
	leakVal := 40
	return &leakVal
}

func RunReturnTest() {
	_ = returnSafeValue()
}

func RunReturnLeakTest() *int {
	return returnLeakyPointer()
}

func main() {
	RunReturnTest()
	leakedPtr := RunReturnLeakTest()
	if leakedPtr != nil {
		MainReturnSink = leakedPtr
	}
}
