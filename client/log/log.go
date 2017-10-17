package log

import (
	"runtime"
	"fmt"
	"time"
)

func init() {
	fmt.Println("will log stuff!")
}

func Func(params ...interface{}) {
	pc, _, _, ok := runtime.Caller(1)
	if !ok {
		fmt.Println(append([]interface{}{"poop"}, params...)...)
	}

	ts := time.Now().Format(time.Stamp)
	fn := runtime.FuncForPC(pc).Name()

	fmt.Printf("%s | %s called : ", ts, fn)
	fmt.Println(params...)
}

func Return(params ...interface{}) {
	pc, _, _, ok := runtime.Caller(1)
	if !ok {
		fmt.Println(append([]interface{}{"poop"}, params...)...)
	}

	ts := time.Now().Format(time.Stamp)
	fn := runtime.FuncForPC(pc).Name()

	fmt.Printf("%s | %s returned: ", ts, fn)
	fmt.Println(params...)
}