package runtime

import (
	"os"
	"os/signal"
	"fmt"
)

var ch chan os.Signal
var cleaners []func(os.Signal)

func init() {
	ch = make(chan os.Signal)

	//TODO: listen to other signals
	//capture and redirect signals
	signal.Notify(ch, os.Interrupt)

	//listen to signals
	go func() {
		sig :=<- ch
		doClean(sig)
	}()
}

func doClean(sig os.Signal) {
	fmt.Println("cleaning")

	for _, v := range cleaners {
		v(sig)
	}

	fmt.Println("exiting")
	os.Exit(1)
}

func AddCleaner(f func(os.Signal)) {
	cleaners = append(cleaners, f)
}

//used only for (programmatically) clean exits
//ie not a panic, or user signal
//does the same thing as do clean but always passes sigint to cleaners
func Exit() {
	sig := os.Interrupt

	for _, v := range cleaners {
		v(sig)
	}

	os.Exit(0)
}

