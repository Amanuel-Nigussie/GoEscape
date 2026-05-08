package main

var GlobalDoublePtr **int

func RunDeepPointerTest() {
	safeVal1 := 170
	safeVal2 := &safeVal1
	safeVal3 := &safeVal2
	_ = safeVal3

	leakVal1 := 180
	leakVal2 := &leakVal1
	GlobalDoublePtr = &leakVal2
}

func main() {
	mainVal := 500
	p1 := &mainVal
	p2 := &p1
	if **p2 == 500 {
		RunDeepPointerTest()
	}
}
