package log

import (
	"runtime"
	"fmt"
	"time"
)

func init() {
	fmt.Println("will log stuff!")
}

func Func(name string, params ...interface{}) {
	pc, _, _, ok := runtime.Caller(1)
	if !ok {
		fmt.Println(append([]interface{}{"poop"}, params...)...)
	}

	ts := time.Now().Format(time.Stamp)
	fn := runtime.FuncForPC(pc).Name()

	if name == "" {
		name = "/"
	}
	fmt.Printf("%s | %s called : %s ", ts, fn, name)
	fmt.Println(params...)
}

func Return(params ...interface{}) {
	ts := time.Now().Format(time.Stamp)

	pc, _, _, ok := runtime.Caller(2)
	if !ok {
		fmt.Printf("%s | _ returned: ", ts)
		fmt.Println(params...)
	}

	fn := runtime.FuncForPC(pc).Name()

	fmt.Printf("%s | %s returned: ", ts, fn)
	fmt.Println(params...)
}