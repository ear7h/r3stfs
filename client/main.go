// Copyright 2017 Julio. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

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



func main() {
	defer runtime.Exit()

	flag.Parse()

	if len(flag.Args()) < 1 {
		log.Fatal("Usage:\n  hello MOUNTPOINT")
	}

	mtpt := flag.Arg(0)


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
