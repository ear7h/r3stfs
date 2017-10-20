package main

import (
	"flag"
	"log"
	"os"

	"github.com/hanwen/go-fuse/fuse/nodefs"
	"github.com/hanwen/go-fuse/fuse/pathfs"

	"r3stfs/client/runtime"
	"fmt"
)

var G_LOCAL_DIR, G_REMOTE_DOMAIN string


func main() {
	defer runtime.Exit()

	flag.Parse()

	if len(flag.Args()) < 1 {
		log.Fatal("Usage:\n  hello MOUNTPOINT")
	}

	mtpt := flag.Arg(0)

	// local dir
	G_LOCAL_DIR = "ear7h_cache/"
	err := os.MkdirAll(G_LOCAL_DIR, 0700)
	if err != nil {
		panic(err)
	}

	G_REMOTE_DOMAIN = "localhost:8080"


	fmt.Println("logged in")
	//make pathfs
	nfs := pathfs.NewPathNodeFs(
		NewR3stFs("localhost:8080", "user", ""),
		&pathfs.PathNodeFsOptions{false, true})

	server, _, err := nodefs.MountRoot(mtpt, nfs.Root(), nil)
	if err != nil {
		fmt.Printf("Mount fail: %v\n", err)
		panic(err)
	}

	fmt.Println("mounted")

	//cleanup
	runtime.AddCleaner(func(signal os.Signal) {
		server.Unmount()
		os.Remove(mtpt)
	})

	server.Serve()
}
