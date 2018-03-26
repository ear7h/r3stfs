package main

import (
	"fmt"

	"github.com/ear7h/r3stfs/server"
)

func main() {
	fmt.Println("server starting")

	err := server.ServeFs(":8080", "", "./store", &server.R3stFsHandler{})
	panic(err)
}