package main

var GlobalIntSink *int
var GlobalValue int

func RunGlobalSafe() {
	safeVal := 50
	GlobalValue = safeVal
}

func RunGlobalLeaky() {
	leakVal := 60
	GlobalIntSink = &leakVal
}

func main() {
	GlobalValue = 100
	RunGlobalSafe()
	if GlobalValue == 50 {
		RunGlobalLeaky()
	}
}
