// Copyright 2016 the Go-FUSE Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"os"
	"sync"
	"syscall"

	"github.com/hanwen/go-fuse/fuse"
	"github.com/hanwen/go-fuse/fuse/nodefs"
	"r3stfs/client/remote"
)

// LoopbackFile delegates all operations back to an underlying os.File.
func NewLoopbackFile(f *os.File) nodefs.File {
	return &loopback{File: f}
}

type loopback struct {
	File *os.File

	// os.File is not threadsafe. Although fd themselves are
	// constant during the lifetime of an open file, the OS may
	// reuse the fd number after it is closed. When open races
	// with another close, they may lead to confusion as which
	// file gets written in the end.
	lock sync.Mutex
}

func (f *loopback) InnerFile() nodefs.File {
	return nil
}

func (f *loopback) SetInode(n *nodefs.Inode) {
}

func (f *loopback) String() string {
	return fmt.Sprintf("loopback(%s)", f.File.Name())
}

func (f *loopback) Read(buf []byte, off int64) (res fuse.ReadResult, code fuse.Status) {
	f.lock.Lock()
	// This is not racy by virtue of the kernel properly
	// synchronizing the open/write/close.
	r := fuse.ReadResultFd(f.File.Fd(), off, len(buf))
	f.lock.Unlock()
	return r, fuse.OK
}

func (f *loopback) Write(data []byte, off int64) (uint32, fuse.Status) {
	fmt.Println("writing: ", string(data))

	f.lock.Lock()
	n, err := f.File.WriteAt(data, off)
	if err != nil {
		fmt.Println("ERROR WRITING: ", err)
	}
	f.lock.Unlock()
	return uint32(n), fuse.ToStatus(err)
}

func (f *loopback) Release() {

	//close file
	f.lock.Lock()
	f.File.Close()
	f.lock.Unlock()

	//send it
	name := f.File.Name()
	fileToSend, err := os.OpenFile(name, os.O_RDONLY, 0700)
	if err != nil {
		return
	}
	stat, err := fileToSend.Stat()
	if err != nil {
		fmt.Println("ayy, ", err)
	}

	fmt.Println("filename: ", name[len(localPath("")):])

	_, err = remote.Put(name[len(localPath("")):], fileToSend, stat.Mode())
	if err != nil {
		fmt.Println(err)
	}
}

func (f *loopback) Flush() fuse.Status {
	f.lock.Lock()

	// Since Flush() may be called for each dup'd fd, we don't
	// want to really close the file, we just want to flush. This
	// is achieved by closing a dup'd fd.
	newFd, err := syscall.Dup(int(f.File.Fd()))
	f.lock.Unlock()

	if err != nil {
		return fuse.ToStatus(err)
	}
	err = syscall.Close(newFd)
	return fuse.ToStatus(err)
}

func (f *loopback) Fsync(flags int) (code fuse.Status) {
	f.lock.Lock()
	r := fuse.ToStatus(syscall.Fsync(int(f.File.Fd())))
	f.lock.Unlock()

	return r
}

func (f *loopback) Flock(flags int) fuse.Status {
	f.lock.Lock()
	r := fuse.ToStatus(syscall.Flock(int(f.File.Fd()), flags))
	f.lock.Unlock()

	return r
}

func (f *loopback) Truncate(size uint64) fuse.Status {
	f.lock.Lock()
	r := fuse.ToStatus(syscall.Ftruncate(int(f.File.Fd()), int64(size)))
	f.lock.Unlock()

	return r
}

func (f *loopback) Chmod(mode uint32) fuse.Status {
	f.lock.Lock()
	r := fuse.ToStatus(f.File.Chmod(os.FileMode(mode)))
	f.lock.Unlock()

	return r
}

func (f *loopback) Chown(uid uint32, gid uint32) fuse.Status {
	f.lock.Lock()
	r := fuse.ToStatus(f.File.Chown(int(uid), int(gid)))
	f.lock.Unlock()

	return r
}

func (f *loopback) GetAttr(a *fuse.Attr) fuse.Status {
	st := syscall.Stat_t{}
	f.lock.Lock()
	err := syscall.Fstat(int(f.File.Fd()), &st)
	f.lock.Unlock()
	if err != nil {
		return fuse.ToStatus(err)
	}
	a.FromStat(&st)

	return fuse.OK
}

// Utimens implemented in files_linux.go and files_darwin.go

// Allocate implemented in files_linux.go and files_darwin.go
