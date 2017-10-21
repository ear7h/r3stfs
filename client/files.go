// Copyright 2017 Julio. All rights reserved.
// Use of this source code is governed by the MIT
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
	"r3stfs/client/log"
)

// LoopbackFile delegates all operations back to an underlying os.file.
func NewLoopbackFile(f *os.File, restPath string, client *remote.Client) nodefs.File {
	return &loopback{
		file: f,
		restPath: restPath,
		remote: client,
	}
}

type loopback struct {
	file   *os.File
	restPath string //path passed in urls
	remote *remote.Client

	// os.file is not threadsafe. Although fd themselves are
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
	return fmt.Sprintf("loopback(%s)", f.file.Name())
}

func (f *loopback) Read(buf []byte, off int64) (res fuse.ReadResult, status fuse.Status) {
	log.Func(f.restPath)
	defer func() {
		log.Return(res, status)
	}()

	f.lock.Lock()
	// This is not racy by virtue of the kernel properly
	// synchronizing the open/write/close.
	r := fuse.ReadResultFd(f.file.Fd(), off, len(buf))
	f.lock.Unlock()
	return r, fuse.OK
}

func (f *loopback) Write(data []byte, off int64) (uint32, fuse.Status) {
	fmt.Println("writing: ", string(data))

	f.lock.Lock()
	n, err := f.file.WriteAt(data, off)
	if err != nil {
		fmt.Println("ERROR WRITING: ", err)
	}
	f.lock.Unlock()
	return uint32(n), fuse.ToStatus(err)
}

func (f *loopback) Release() {
	log.Func(f.restPath)
	defer func() {
		log.Return()
	}()

	//close file
	f.lock.Lock()
	f.file.Close()
	f.lock.Unlock()

	name := f.file.Name()

	fileToSend, err := os.OpenFile(name, os.O_RDONLY, 0600)
	if err != nil {
		fmt.Println("err", err)
		return
	}

	fmt.Println("filename: ", f.restPath)

	_, err = f.remote.Put(f.restPath, fileToSend)
	if err != nil {
		fmt.Println(err)
	}
}

func (f *loopback) Flush() (status fuse.Status) {
	log.Func(f.restPath)
	defer func() {
		log.Return(status)
	}()


	f.lock.Lock()

	// Since Flush() may be called for each dup'd fd, we don't
	// want to really close the file, we just want to flush. This
	// is achieved by closing a dup'd fd.
	newFd, err := syscall.Dup(int(f.file.Fd()))
	f.lock.Unlock()

	if err != nil {
		return fuse.ToStatus(err)
	}
	err = syscall.Close(newFd)
	return fuse.ToStatus(err)
}

func (f *loopback) Fsync(flags int) (code fuse.Status) {
	f.lock.Lock()
	r := fuse.ToStatus(syscall.Fsync(int(f.file.Fd())))
	f.lock.Unlock()

	return r
}

func (f *loopback) Flock(flags int) fuse.Status {
	f.lock.Lock()
	r := fuse.ToStatus(syscall.Flock(int(f.file.Fd()), flags))
	f.lock.Unlock()

	return r
}

func (f *loopback) Truncate(size uint64) fuse.Status {
	f.lock.Lock()
	r := fuse.ToStatus(syscall.Ftruncate(int(f.file.Fd()), int64(size)))
	f.lock.Unlock()

	return r
}

func (f *loopback) Chmod(mode uint32) fuse.Status {
	f.lock.Lock()
	r := fuse.ToStatus(f.file.Chmod(os.FileMode(mode)))
	f.lock.Unlock()

	return r
}

func (f *loopback) Chown(uid uint32, gid uint32) fuse.Status {
	f.lock.Lock()
	r := fuse.ToStatus(f.file.Chown(int(uid), int(gid)))
	f.lock.Unlock()

	return r
}

func (f *loopback) GetAttr(a *fuse.Attr) fuse.Status {
	st := syscall.Stat_t{}
	f.lock.Lock()
	err := syscall.Fstat(int(f.file.Fd()), &st)
	f.lock.Unlock()
	if err != nil {
		return fuse.ToStatus(err)
	}
	a.FromStat(&st)

	return fuse.OK
}

// Utimens implemented in files_linux.go and files_darwin.go

// Allocate implemented in files_linux.go and files_darwin.go
