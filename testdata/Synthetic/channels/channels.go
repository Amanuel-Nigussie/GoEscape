package main

var pointerChan = make(chan *int)
var valueChan = make(chan int)

func RunChannelSafe() {
	safeVal := 130
	go func() { valueChan <- safeVal }()
}

func RunChannelLeaky() {
	leakVal := 140
	go func() { pointerChan <- &leakVal }()
}

func main() {
	select {
	case <-valueChan:
	default:
		RunChannelSafe()
		RunChannelLeaky()
	}
}