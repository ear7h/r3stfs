package main

import (
	"testing"
	"github.com/hanwen/go-fuse/fuse/pathfs"
	"github.com/hanwen/go-fuse/fuse/nodefs"
	"fmt"
	"os"
	"r3stfs/client/runtime"
)

func TestR3stFs_cacheOK(t *testing.T) {
	mtpt := "testfs"
	rfs := NewR3stFs("localhost:8080", "user", "")

	//make pathfs
	nfs := pathfs.NewPathNodeFs(
		rfs,
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

	go server.Serve()

	//the test
	if rfs.cacheOK("hello.go") {
		t.Errorf("cache should not be ok")
	}

	fmt.Printf("\n\n%v\n\n", rfs.cacheOK("hello.go"))

	runtime.Exit()
}