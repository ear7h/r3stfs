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
	"time"
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
		if status != fuse.OK {
			log.Return("ERROR", res, status)
		} else {
			log.Return(res, status)
		}
	}()

	f.lock.Lock()
	// This is not racy by virtue of the kernel properly
	// synchronizing the open/write/close.
	r := fuse.ReadResultFd(f.file.Fd(), off, len(buf))
	f.lock.Unlock()
	return r, fuse.OK
}

func (f *loopback) Write(data []byte, off int64) (n uint32, status fuse.Status) {
	log.Func(f.restPath)
	defer func() {
		if status != fuse.OK {
			log.Return("ERROR", n, status)
		} else {
			log.Return(n, status)
		}
	}()
	fmt.Println("writing: ", string(data))

	f.lock.Lock()
	nint, err := f.file.WriteAt(data, off)
	if err != nil {
		fmt.Println("ERROR WRITING: ", err)
	}
	f.lock.Unlock()
	n, status = uint32(nint), fuse.ToStatus(err)
	return
}

func (f *loopback) Release() {
	status := fuse.OK

	log.Func(f.restPath)
	defer func() {
		if status != fuse.OK {
			log.Return("ERROR")
		} else {
			log.Return()
		}
	}()

	//close file
	f.lock.Lock()
	f.file.Close()
	f.lock.Unlock()

	name := f.file.Name()

	fileToSend, err := os.OpenFile(name, os.O_RDONLY, 0600)
	if err != nil {
		fmt.Println("err", err)
		status = fuse.ToStatus(err)
		return
	}

	fmt.Println("filename: ", f.restPath)

	_, err = f.remote.Put(f.restPath, fileToSend)
	if err != nil {
		fmt.Println(err)
		status = fuse.ToStatus(err)
	}
}

func (f *loopback) Flush() (status fuse.Status) {
	log.Func(f.restPath)
	defer func() {
		if status != fuse.OK {
			log.Return("ERROR", status)
		} else {
			log.Return(status)
		}
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

func (f *loopback) Fsync(flags int) (status fuse.Status) {
	log.Func(f.restPath)
	defer func() {
		if status != fuse.OK {
			log.Return("ERROR", status)
		} else {
			log.Return(status)
		}
	}()

	f.lock.Lock()
	status = fuse.ToStatus(syscall.Fsync(int(f.file.Fd())))
	f.lock.Unlock()

	return
}

func (f *loopback) Flock(flags int) (status fuse.Status) {
	log.Func(f.restPath)
	defer func() {
		if status != fuse.OK {
			log.Return("ERROR", status)
		} else {
			log.Return(status)
		}
	}()

	f.lock.Lock()
	status = fuse.ToStatus(syscall.Flock(int(f.file.Fd()), flags))
	f.lock.Unlock()

	return
}

func (f *loopback) Truncate(size uint64) (status fuse.Status) {
	log.Func(f.restPath)
	defer func() {
		if status != fuse.OK {
			log.Return("ERROR", status)
		} else {
			log.Return(status)
		}
	}()

	f.lock.Lock()
	status = fuse.ToStatus(syscall.Ftruncate(int(f.file.Fd()), int64(size)))
	f.lock.Unlock()

	return
}

func (f *loopback) Chmod(mode uint32) (status fuse.Status) {
	log.Func(f.restPath)
	defer func() {
		if status != fuse.OK {
			log.Return("ERROR", status)
		} else {
			log.Return(status)
		}
	}()

	f.lock.Lock()
	status = fuse.ToStatus(f.file.Chmod(os.FileMode(mode)))
	f.lock.Unlock()

	return
}

func (f *loopback) Chown(uid uint32, gid uint32) (status fuse.Status) {
	log.Func(f.restPath)
	defer func() {
		if status != fuse.OK {
			log.Return("ERROR", status)
		} else {
			log.Return(status)
		}
	}()


	f.lock.Lock()
	status = fuse.ToStatus(f.file.Chown(int(uid), int(gid)))
	f.lock.Unlock()

	return
}

func (f *loopback) GetAttr(a *fuse.Attr) (status fuse.Status) {
	log.Func(f.restPath)
	defer func() {
		if status != fuse.OK {
			log.Return("ERROR", status)
		} else {
			log.Return(status)
		}
	}()

	st := syscall.Stat_t{}
	f.lock.Lock()
	err := syscall.Fstat(int(f.file.Fd()), &st)
	f.lock.Unlock()
	if err != nil {
		status = fuse.ToStatus(err)
		return
	}
	a.FromStat(&st)
	status = fuse.OK

	return
}

// Utimens - file handle based version of loopbackFileSystem.Utimens()
func (f *loopback) Utimens(a *time.Time, m *time.Time) (status fuse.Status) {
	//TODO implement server side attr changes
	status = fuse.ToStatus(os.Chtimes(f.file.Name(), *a, *m))
	return
}


// Allocate implemented in files_linux.go and files_darwin.go
