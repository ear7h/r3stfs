// Copyright 2016 the Go-FUSE Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// A Go mirror of libfuse's hello.c

package main

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/hanwen/go-fuse/fuse"
	"github.com/hanwen/go-fuse/fuse/nodefs"
	"github.com/hanwen/go-fuse/fuse/pathfs"

	"r3stfs/client/log"
	"r3stfs/client/remote"
)

type R3stFs struct {
	pathfs.FileSystem
	client *remote.Client
}

func (rfs *R3stFs) GetAttr(name string, context *fuse.Context) (attr *fuse.Attr, status fuse.Status) {
	log.Func(name, context)
	defer log.Return(attr, status)

	if cacheOK(name, rfs.client) {
		var err error
		sysStat := syscall.Stat_t{}

		if name == "" {
			err = syscall.Stat(localPath(name), &sysStat)
		} else {
			err = syscall.Lstat(localPath(name), &sysStat)
		}

		if err != nil {
			fmt.Println("err")
			attr, status = nil, fuse.ToStatus(err)
			return
		}

		attr, status = &fuse.Attr{}, fuse.OK
		attr.FromStat(&sysStat)

		return
	}

	resp, err := rfs.client.Head(name)
	if err != nil {
		attr, status = nil, fuse.ToStatus(err)
		return
	}
	if resp.StatusCode == http.StatusNotFound {
		attr, status = nil, fuse.ENOENT
		return
	}
	if resp.StatusCode != http.StatusOK {
		attr, status = nil, fuse.EAGAIN
		return
	}

	if resp.Header["Is-Dir"][0] == "true" {
		mode, err := strconv.ParseInt(resp.Header["file-Mode"][0], 8, 32)
		if err != nil {
			mode = 0
		}

		mode |= fuse.S_IFDIR

		modTime, err := time.Parse(time.RFC1123, strings.Join(resp.Header["Mode"], " "))

		//return dir
		attr, status = &fuse.Attr{
			Mtime: uint64(modTime.Unix()),
			Mode:  uint32(mode),
		}, fuse.OK
		return
	}

	size, err := strconv.ParseInt(resp.Header["Content-Length"][0], 10, 64)
	if err != nil {
		size = 0
	}

	mode, err := strconv.ParseInt(resp.Header["file-Mode"][0], 8, 32)
	if err != nil {
		mode = 0
	}

	mode |= fuse.S_IFREG

	modTime, err := time.Parse(time.RFC1123, strings.Join(resp.Header["Mode"], " "))

	//return regular file
	attr, status = &fuse.Attr{
		Size:  uint64(size),
		Mtime: uint64(modTime.Unix()),
		Mode:  uint32(mode),
	}, fuse.OK

	return
}

func (rfs *R3stFs) OpenDir(name string, context *fuse.Context) (dir []fuse.DirEntry, status fuse.Status) {
	log.Func(name, context)
	defer log.Return(dir, status)

	resp, err := rfs.client.Get(name)
	if err != nil {
		dir, status = nil, fuse.ToStatus(err)
		return
	}
	if resp.StatusCode == http.StatusNotFound {
		dir, status = nil, fuse.ENOENT
	}
	if resp.StatusCode != http.StatusOK {
		dir, status = nil, fuse.EAGAIN
	}

	scanner := bufio.NewScanner(resp.Body)
	scanner.Split(bufio.ScanLines)

	for scanner.Scan() {
		str := scanner.Text()

		arr := strings.Split(str, " ")

		if len(arr) != 2 {
			return nil, fuse.EAGAIN
		}

		i, err := strconv.ParseInt(arr[1], 8, 0)
		if err != nil {
			return nil, fuse.EAGAIN
		}

		if i&int64(os.ModeDir) > 0 {
			err = os.MkdirAll(localPath(arr[0]), os.FileMode(i))
			if err != nil {
				fmt.Println("err: ", err)
			}
		}

		dir = append(dir, fuse.DirEntry{Name: arr[0], Mode: uint32(i)})
	}

	status = fuse.OK
	return
}

func (rfs *R3stFs) Open(name string, flags uint32, context *fuse.Context) (file nodefs.File, status fuse.Status) {
	log.Func(name, flags, context)
	defer log.Return(file, status)

use_cache:
	if cacheOK(name, rfs.client) {
		f, err := os.OpenFile(localPath(name), int(flags), 0)
		if err != nil {
			fmt.Println("open err: ", err)
		}
		file, status = NewLoopbackFile(f, rfs.client), fuse.OK
		return

	}

	//get file remotely
	resp, err := rfs.client.Get(name)
	if err != nil {
		file, status = nil, fuse.ToStatus(err)
		return
	}

	f, err := os.OpenFile(localPath(name), os.O_CREATE|os.O_WRONLY, 0700)
	if err != nil {
		file, status = nil, fuse.ToStatus(err)
		return
	}

	b, err := io.Copy(f, resp.Body)
	if err != nil {
		file, status = nil, fuse.ToStatus(err)
		return
	}

	// close readers/writers
	resp.Body.Close()
	f.Close()

	fmt.Printf("%d bytes written locally", b)

	//file downloaded successfully
	goto use_cache
}

func (rfs *R3stFs) Create(name string, flags uint32, mode uint32, context *fuse.Context) (file nodefs.File, status fuse.Status) {
	log.Func(name, flags, mode, context)
	defer log.Return(file, status)

	f, err := os.OpenFile(localPath(name), os.O_CREATE|int(flags), os.FileMode(mode))
	if err != nil {
		file, status = nil, fuse.ToStatus(err)
		return
	}

	_, err = rfs.client.Put(name, f)
	if err != nil {
		file, status = nil, fuse.ToStatus(err)
		return
	}

	file, status = NewLoopbackFile(f, rfs.client), fuse.OK
	return
}

func (rfs *R3stFs) Rename(oldName string, newName string, context *fuse.Context) (status fuse.Status) {
	log.Func(oldName, newName, context)
	defer log.Return(status)

	//send message to server

	f, err := os.OpenFile(localPath(oldName), os.O_RDONLY, 0700)
	if err != nil {
		status = fuse.ToStatus(err)
		return
	}

	_, err = rfs.client.Put(newName, f)
	if err != nil {
		status = fuse.ToStatus(err)
		return
	}

	_, err = rfs.client.Delete(oldName)
	if err != nil {
		status = fuse.ToStatus(err)
		return
	}

	err = os.Rename(localPath(oldName), localPath(newName))
	if err != nil {
		status = fuse.ToStatus(err)
		return
	}

	status = fuse.OK
	return
}

func (rfs *R3stFs) Unlink(name string, context *fuse.Context) (code fuse.Status) {
	return fuse.ToStatus(syscall.Unlink(localPath(name)))
}

func (rfs *R3stFs) Rmdir(name string, context *fuse.Context) (code fuse.Status) {
	return fuse.ToStatus(syscall.Rmdir(localPath(name)))
}

func (rfs *R3stFs) Access(name string, mode uint32, context *fuse.Context) fuse.Status {
	return fuse.ToStatus(syscall.Access(localPath(name), mode))
}

func (rfs *R3stFs) Truncate(path string, offset uint64, context *fuse.Context) (code fuse.Status) {
	return fuse.ToStatus(os.Truncate(localPath(path), int64(offset)))
}
