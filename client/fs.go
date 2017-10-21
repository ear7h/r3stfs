// Copyright 2017 Julio. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.


package main

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/0xAX/notificator"
	"github.com/hanwen/go-fuse/fuse"
	"github.com/hanwen/go-fuse/fuse/nodefs"
	"github.com/hanwen/go-fuse/fuse/pathfs"

	"r3stfs/client/log"
	"r3stfs/client/remote"
	"r3stfs/sandbox"
)

type R3stFs struct {
	pathfs.FileSystem
	client *remote.Client
	cache  sandbox.Store
}

func (rfs *R3stFs) cacheOK(name string) bool {
	notify := func(err error) {
		fmt.Println("cacheOK err: ", err)

		e := notificator.New(notificator.Options{
			AppName: "r3stfs",
		}).Push("Cache Error", fmt.Sprint(err), "", notificator.UR_NORMAL)

		if e != nil {
			fmt.Println("cacheOK: notify - ", e)
		}
	}


	resp, err := rfs.client.Head(name)
	if err != nil {
		go notify(err)
		return false
	}

	//if not exist
	if resp.StatusCode == http.StatusNotFound {
		rfs.cache.RemoveAll(name)
		return true
	}

	remoteUnix, err := strconv.ParseInt(resp.Header.Get("Mtime"), 10, 64)
	if err != nil {
		go notify(err)
		return true
	}

	stat, err := rfs.cache.Stat(name)
	if err != nil {
		go notify(err)
		return false
	}

	// cache is outdated
	if time.Unix(remoteUnix, 0).After(stat.ModTime()) {
		fmt.Println("cache miss")
		return false
	}

	return true
}

func (rfs *R3stFs) GetAttr(name string, context *fuse.Context) (attr *fuse.Attr, status fuse.Status) {
	log.Func(name, context)
	defer func() {
		log.Return(attr, status)
	}()

	if rfs.cacheOK(name) {
		var err error
		sysStat := syscall.Stat_t{}

		if name == "" {
			err = syscall.Stat(rfs.cache.Abs(name), &sysStat)
		} else {
			err = syscall.Lstat(rfs.cache.Abs(name), &sysStat)
		}

		if err != nil {
			fmt.Println("err: ", err)
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

	mode, err := strconv.ParseInt(resp.Header["File-Mode"][0], 8, 32)
	if err != nil {
		mode = 0
	}

	mTime, err := strconv.ParseInt(resp.Header["Mtime"][0], 10, 64)
	if err != nil {
		attr, status = nil, fuse.ToStatus(err)
		return
	}

	aTime, err := strconv.ParseInt(resp.Header["Atime"][0], 10, 64)
	if err != nil {
		attr, status = nil, fuse.ToStatus(err)
		return
	}

	rfs.cache.Chtimes(name, time.Unix(aTime, 0), time.Unix(mTime, 0))

	if resp.Header["Is-Dir"][0] == "true" {
		mode |= syscall.S_IFDIR

		//return dir
		attr, status = &fuse.Attr{
			Mtime: uint64(mTime),
			Atime: uint64(aTime),
			Mode:  uint32(mode),
		}, fuse.OK

		return attr, status
	}

	mode |= syscall.S_IFREG

	size, err := strconv.ParseInt(resp.Header["Content-Length"][0], 10, 64)
	if err != nil {
		size = 0
	}

	//return regular file
	attr, status = &fuse.Attr{
		Size:  uint64(size),
		Mtime: uint64(mTime),
		Atime: uint64(aTime),
		Mode:  uint32(mode),
	}, fuse.OK

	return
}

func (rfs *R3stFs) OpenDir(name string, context *fuse.Context) (dir []fuse.DirEntry, status fuse.Status) {
	log.Func(name, context)
	defer func() {
		log.Return(dir, status)
	}()

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

		i, err := strconv.ParseUint(arr[1], 8, 32)
		if err != nil {
			return nil, fuse.EAGAIN
		}

		p := path.Join(name, arr[0])
		fmt.Println("name join arr[0]", p)
		m := os.FileMode(i)

		//make structure and dummy files
		fmt.Println("mode: ", arr[1])
		fmt.Println("mode: ", arr[1])
		fmt.Println("mode: ", arr[1])
		fmt.Println("mode: ", arr[1])
		fmt.Println("mode: ", m&fuse.S_IFDIR)

		if i&syscall.S_IFDIR != 0 {
			fmt.Println("making: ", p)
			err = rfs.cache.MkDirAll(p, os.FileMode(i))
			if err != nil {
				fmt.Println("err: ", err)
			}
		} else {
			fmt.Println("making: ", p)
			f, err := rfs.cache.OpenFile(p, os.O_CREATE|os.O_RDONLY, os.FileMode(i))
			if err != nil {
				fmt.Println("err: ", err)
			}
			f.Close()
		}

		rfs.cache.Chtimes(p, time.Time{}, time.Time{})

		dir = append(dir, fuse.DirEntry{Name: arr[0], Mode: uint32(i)})
	}

	status = fuse.OK
	return
}

func (rfs *R3stFs) Open(name string, flags uint32, context *fuse.Context) (file nodefs.File, status fuse.Status) {
	log.Func(name, flags, context)
	defer func() {
		log.Return(file, status)
	}()

use_cache:
	if rfs.cacheOK(name) {
		f, err := rfs.cache.OpenFile(name, int(flags), 0)
		if err != nil {
			fmt.Println("open err: ", err)
		}
		file, status = NewLoopbackFile(f, name, rfs.client), fuse.OK
		return

	}

	//get file remotely
	resp, err := rfs.client.Get(name)
	if err != nil {
		file, status = nil, fuse.ToStatus(err)
		return
	}
	perm, err := strconv.ParseUint(resp.Header.Get("File-Mode"), 8, 32)
	if err != nil {
		file, status = nil, fuse.ToStatus(err)
		return
	}

	//open the file using the requested permission bits
	f, err := rfs.cache.OpenFile(name, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(perm))
	if err != nil {
		fmt.Println("err: ", err)
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
	defer func() {
		log.Return(file, status)
	}()

	// create and close to register in host file system
	f, err := rfs.cache.OpenFile(name, os.O_CREATE, os.FileMode(mode))
	if err != nil {
		file, status = nil, fuse.ToStatus(err)
		return
	}

	_, err = rfs.client.Put(name, f)
	if err != nil {
		file, status = nil, fuse.ToStatus(err)
		return
	}

	f.Close()

	f, err = rfs.cache.OpenFile(name, int(flags), os.FileMode(mode))
	if err != nil {
		file, status = nil, fuse.ToStatus(err)
		return
	}


	file, status = NewLoopbackFile(f, name, rfs.client), fuse.OK
	return
}

func (rfs *R3stFs) Rename(oldName string, newName string, context *fuse.Context) (status fuse.Status) {
	log.Func(oldName, newName, context)
	defer func() {
		log.Return(status)
	}()

	//send message to server

	f, err := rfs.cache.OpenFile(oldName, os.O_RDONLY, 0700)
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

	err = rfs.cache.Rename(oldName, newName)
	if err != nil {
		status = fuse.ToStatus(err)
		return
	}

	status = fuse.OK
	return
}

func (rfs *R3stFs) Unlink(name string, context *fuse.Context) (status fuse.Status) {
	log.Func(name, context)
	defer func() {
		log.Return(status)
	}()

	_, err := rfs.client.Delete(name)
	if err != nil {
		status = fuse.ToStatus(err)
	}

	status = fuse.ToStatus(syscall.Unlink(rfs.cache.Abs(name)))
	return
}

func (rfs *R3stFs) Rmdir(name string, context *fuse.Context) (status fuse.Status) {
	log.Func(name, context)
	defer func() {
		log.Return(status)
	}()

	_, err := rfs.client.Delete(name)
	if err != nil {
		status = fuse.ToStatus(err)
		return
	}

	status = fuse.ToStatus(syscall.Rmdir(rfs.cache.Abs(name)))
	return
}

// Access can always use a cached file as the structure is
// always updated in the OpenDir call
func (rfs *R3stFs) Access(name string, mode uint32, context *fuse.Context) (status fuse.Status) {
	log.Func(name, context)
	defer func() {
		log.Return(status)
	}()

	status = fuse.ToStatus(syscall.Access(rfs.cache.Abs(name), mode))
	return
}

func (rfs *R3stFs) Truncate(name string, offset uint64, context *fuse.Context) (status fuse.Status) {
	log.Func(name, context)
	defer func() {
		log.Return(status)
	}()

	err := os.Truncate(rfs.cache.Abs(name), int64(offset))
	if err != nil {
		status = fuse.ToStatus(err)
		return
	}

	f, err := rfs.cache.OpenFile(name, os.O_RDONLY, 0200)
	if err != nil {
		status = fuse.ToStatus(err)
		return
	}

	_, err = rfs.client.Put(name, f)
	if err != nil {
		status = fuse.ToStatus(err)
		return
	}

	status = fuse.OK
	return
}

func NewR3stFs(host, user, pass string) *R3stFs {
	client := remote.Login(host, user, pass)

	//u, err := osuser.Current()
	//if err != nil {
	//	panic(err)
	//}
	//
	//homedir := u.HomeDir
	//
	//cache, err := sandbox.NewStore(path.Join(homedir, host, user))
	cache, err := sandbox.NewStore("ear7h_cache/localhost:8080")
	if err != nil {
		panic(err)
	}

	return &R3stFs{
		FileSystem: pathfs.NewDefaultFileSystem(),
		client:     client,
		cache:      cache,
	}
}
