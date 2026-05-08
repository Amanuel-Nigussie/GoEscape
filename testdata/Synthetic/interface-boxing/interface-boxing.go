package main

var GlobalAny interface{}

func RunInterfaceSafe() {
	safeVal := 90
	GlobalAny = safeVal
}

func RunInterfaceLeaky() {
	leakVal := 100
	GlobalAny = &leakVal
}

func main() {
	var pipeline []interface{}
	pipeline = append(pipeline, 1)
	if len(pipeline) > 0 {
		RunInterfaceSafe()
		RunInterfaceLeaky()
	}
}
