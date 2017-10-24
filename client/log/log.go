package log

import (
	"runtime"
	"fmt"
	"time"
	"os"
	"io"
)

var outStream io.Writer

func init() {
	fmt.Println("will log stuff!")
	outFile, err := os.OpenFile("log.txt", os.O_CREATE | os.O_TRUNC | os.O_WRONLY, 0666)
	if err != nil {
		panic(err)
	}
	outStream = io.MultiWriter(os.Stdout, outFile)

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
	fmt.Fprintf(outStream, "%s | %s called : %s ", ts, fn, name)
	fmt.Fprintln(outStream, params...)
}

func Return(params ...interface{}) {
	ts := time.Now().Format(time.Stamp)

	pc, _, _, ok := runtime.Caller(2)
	if !ok {
		fmt.Printf("%s | _ returned: ", ts)
		fmt.Println(params...)
	}

	fn := runtime.FuncForPC(pc).Name()

	fmt.Fprintf(outStream,"%s | %s returned: ", ts, fn)
	fmt.Fprintln(outStream, params...)
}