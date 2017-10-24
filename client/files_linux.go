// Copyright 2016 the Go-FUSE Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"syscall"
	"time"

	"github.com/hanwen/go-fuse/fuse"
)

func (f *loopback) Allocate(off uint64, sz uint64, mode uint32) fuse.Status {
	//TODO: empty POST request

	f.lock.Lock()
	err := syscall.Fallocate(int(f.file.Fd()), mode, int64(off), int64(sz))
	f.lock.Unlock()
	if err != nil {
		return fuse.ToStatus(err)
	}
	return fuse.OK
}