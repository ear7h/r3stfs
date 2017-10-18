package main

import (
	"flag"
	"log"
	"os"

	"github.com/hanwen/go-fuse/fuse/nodefs"
	"github.com/hanwen/go-fuse/fuse/pathfs"

	"r3stfs/client/remote"
	"r3stfs/client/runtime"
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

	//setup api caller
	client := remote.Login("localhost:8080", "user", "")

	//make pathfs
	nfs := pathfs.NewPathNodeFs(&R3stFs{
		FileSystem: pathfs.NewDefaultFileSystem(),
		client:client,
	}, &pathfs.PathNodeFsOptions{false, false})
	server, _, err := nodefs.MountRoot(mtpt, nfs.Root(), nil)
	if err != nil {
		log.Fatalf("Mount fail: %v\n", err)
	}
	nfs.SetDebug(true)

	//cleanup
	runtime.AddCleaner(func(signal os.Signal) {
		server.Unmount()
		os.Remove(mtpt)
	})

	server.Serve()
}
