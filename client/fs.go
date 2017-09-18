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
	"time"

	"github.com/hanwen/go-fuse/fuse"
	"github.com/hanwen/go-fuse/fuse/nodefs"
	"github.com/hanwen/go-fuse/fuse/pathfs"

	"r3stfs/client/remote"
)

type R3stFs struct {
	pathfs.FileSystem
}

func (rfs *R3stFs) GetAttr(name string, context *fuse.Context) (*fuse.Attr, fuse.Status) {
	//debug
	fmt.Println("get attr: ", name)

	cs := trustCache(name)

	if cs == CACHE_GOOD {
		attr, err := os.Stat(name)

		if err != nil {
			return nil, fuse.ToStatus(err)
		}
		return fuse.ToAttr(attr), fuse.OK
	}

	resp, err := remote.Head(name)
	if err != nil {
		return nil, fuse.ToStatus(err)
	}
	if resp.StatusCode == http.StatusNotFound {
		return nil, fuse.ENOENT
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fuse.EAGAIN
	}

	if resp.Header["Is-Dir"][0] == "true" {
		mode, err := strconv.ParseInt(resp.Header["File-Mode"][0], 8, 32)
		if err != nil {
			mode = 0
		}

		mode |= fuse.S_IFDIR

		modTime, err := time.Parse(time.RFC1123, strings.Join(resp.Header["Mode"], " "))

		//return dir
		return &fuse.Attr{
			Mtime: uint64(modTime.Unix()),
			Mode:  uint32(mode),
		}, fuse.OK
	}

	size, err := strconv.ParseInt(resp.Header["Content-Length"][0], 10, 64)
	if err != nil {
		size = 0
	}

	mode, err := strconv.ParseInt(resp.Header["File-Mode"][0], 8, 32)
	if err != nil {
		mode = 0
	}

	mode |= fuse.S_IFREG

	modTime, err := time.Parse(time.RFC1123, strings.Join(resp.Header["Mode"], " "))

	//return regular file
	return &fuse.Attr{
		Size:  uint64(size),
		Mtime: uint64(modTime.Unix()),
		Mode:  uint32(mode),
	}, fuse.OK
}

func (rfs *R3stFs) OpenDir(name string, context *fuse.Context) (c []fuse.DirEntry, code fuse.Status) {
	//debug
	fmt.Println("open dir:")

	resp, err := remote.Get(name)
	if err != nil {
		return nil, fuse.ToStatus(err)
	}
	if resp.StatusCode == http.StatusNotFound {
		return nil, fuse.ENOENT
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fuse.EAGAIN
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

		c = append(c, fuse.DirEntry{Name: arr[0], Mode: uint32(i)})
	}

	return c, fuse.OK
}

func (rfs *R3stFs) Open(name string, flags uint32, context *fuse.Context) (file nodefs.File, code fuse.Status) {
	//debug
	fmt.Println("open: ", name, " ==========")

	cs := trustCache(name)

	if cs == CACHE_ENOENT {
		return nil, fuse.ENOENT
	}

use_cache:
	if cs == CACHE_GOOD {
		f, err := os.OpenFile(localPath(name), int(flags), 0)
		if err != nil {
			fmt.Println("open err: ", err)
		}
		return NewLoopbackFile(f), fuse.OK
	}

	//get file remotely
	resp, err := remote.Get(name)
	if err != nil {
		return nil, fuse.ToStatus(err)
	}

	f, err := os.OpenFile(localPath(name), os.O_CREATE|os.O_WRONLY, 0700)
	if err != nil {
		return nil, fuse.ToStatus(err)
	}

	b, err := io.Copy(f, resp.Body)
	if err != nil {
		fmt.Println("error making local copy: ",err)
		return nil, fuse.ToStatus(err)
	}


	fmt.Printf("%d bytes written locally", b)

	f.Close()

	//file downloaded successfully
	cs = CACHE_GOOD
	goto use_cache
}

func (rfs *R3stFs) Create(name string, flags uint32, mode uint32, context *fuse.Context) (file nodefs.File, code fuse.Status) {
	f, err := os.OpenFile(localPath(name), os.O_CREATE|int(flags), os.FileMode(mode))
	if err != nil {
		return nil, fuse.ToStatus(err)
	}

	//the loopback file will send a PUT request on release, updating the remote

	return NewLoopbackFile(f), fuse.OK
}

func (rfs *R3stFs) Rename(oldName string, newName string, context *fuse.Context) (code fuse.Status) {
	fmt.Println("rename")
	//send message to server

	f, err := os.OpenFile(localPath(oldName), os.O_RDONLY, 0)
	if err != nil {
		return fuse.ToStatus(err)
	}
	stat, err := f.Stat()
	if err != nil {
		return fuse.ToStatus(err)
	}

	_, err = remote.Put(newName, f, stat.Mode())
	if err != nil {
		return fuse.ToStatus(err)
	}

	_, err = remote.Delete(oldName)
	if err != nil {
		return fuse.ToStatus(err)
	}

	return fuse.ToStatus(os.Rename(localPath(oldName), localPath(newName)))
}



