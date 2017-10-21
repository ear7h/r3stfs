// Copyright 2017 Julio. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package main

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"r3stfs/sandbox"
)

//
// Utility functions
//

//called only by top level handlers
func serveError(w http.ResponseWriter, err error) {
	fmt.Println("error\n", err)

	if os.IsNotExist(err) {
		http.Error(w, "enoent", http.StatusNotFound)
	} else {
		http.Error(w, "oops", http.StatusInternalServerError)
	}

	return
}

//gets file mode from request header
func getMode(r *http.Request) (os.FileMode, error) {
	fmt.Println("header", r.Header)

	i, err := strconv.ParseInt(r.Header.Get("File-Mode"), 8, 32)
	if err != nil {
		return 0, err
	}

	fmt.Println("mode ", os.FileMode(int32(i)))

	return os.FileMode(int32(i)), nil
}

/*
HEAD /dir/file.txt

200
File-Mode: 0777
Is-Dir: false
Last-Modified: Mon, 02 Jan 2006 15:04:05 MST //rfc 1123
Atime: 15324230 // Unix time
Mtime: 15324230 // Unix time
Content-Length: 1000 //in bytes
*/

// write file attributes to passed ResponseWriter's header
//
// writeHead will make one write to the errc channel
// a signal to the caller to proceed with io
// 		if it is not nil don't do io
// and a read
//		this is to hang until the caller is done with io before
//		asserting the old mtime and atime
func writeHead(w http.ResponseWriter, userSpace sandbox.Store, file string, errc chan error, wg *sync.WaitGroup) {
	defer close(errc)

	fi, err := userSpace.Stat(file)
	if err != nil {
		fmt.Println("wh there was error")
		errc <- err
		fmt.Println("wh err sent")
		return
	}
	// signal the caller it is okay to access files

	stat, ok := fi.Sys().(*syscall.Stat_t)
	if !ok {
		fmt.Println("wh there was error")
		errc <- err
		fmt.Println("wh err sent")
	}

	fmt.Println("wh no error")
	errc <- nil
	atime := stat.Atimespec.Sec
	mtime := stat.Mtimespec.Sec

	w.Header().Set("File-Mode", strconv.FormatUint(uint64(stat.Mode), 8))
	w.Header().Set("Is-Dir", strconv.FormatBool(fi.Mode().IsDir()))
	w.Header().Set("Last-Modified", fi.ModTime().Format(time.RFC1123))
	w.Header().Set("Mtime", strconv.FormatInt(mtime, 10))
	w.Header().Set("Atime", strconv.FormatInt(atime, 10))

	if !fi.IsDir() {
		w.Header().Set("Content-Length", strconv.FormatInt(fi.Size(), 10))
	}

	fmt.Println("headers written\n", w.Header())

	fmt.Println("writeHead: waiting send finish")
	wg.Done()
	wg.Wait()
	fmt.Println("writeHead: proceeding")

	// assert old times
	userSpace.Chtimes(file,
		time.Unix(stat.Atimespec.Unix()),
		time.Unix(stat.Atimespec.Unix()))
}

// Authorization:name pass
func auth(header string) (string, bool) {
	arr := strings.Split(header, " ")

	if len(arr) != 2 {
		return "", false
	}

	byt, err := base64.StdEncoding.DecodeString(arr[1])
	if err != nil {
		return "", false
	}

	return string(byt), true
}

//
// Http handlers
//
// receive the two standard http parameters and string of the username
//

// file attributes
func serveHead(w http.ResponseWriter, r *http.Request, userSpace sandbox.Store) {
	//
	c := make(chan error, 1)
	var wg sync.WaitGroup
	wg.Add(2) //wait for writehead and handler to finish
	go writeHead(w, userSpace, r.URL.Path, c, &wg)

	// check if the header was written properly
	err := <-c
	if err != nil {
		fmt.Println("serveHead: error")
		serveError(w, err)
		return
	}

	fmt.Println("serveHead: okay")
	wg.Done()
	wg.Wait()
	fmt.Println("serveHead: returning")
}

//read files
func serveGet(w http.ResponseWriter, r *http.Request, userSpace sandbox.Store) {

	requestPath := r.URL.Path

	c := make(chan error, 1)
	var wg sync.WaitGroup
	wg.Add(2) //wait for writehead and handler to finish
	go writeHead(w, userSpace, requestPath, c, &wg)

	//check if io is okay
	fmt.Println("get checking error")
	err := <-c
	if err != nil {
		fmt.Println("get there was error")
		serveError(w, err)
		return
	}

	f, err := userSpace.Open(requestPath)
	if err != nil {
		serveError(w, err)
		return
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		serveError(w, err)
		return
	}

	if stat.IsDir() {
		dir, err := f.Readdir(0)
		if err != nil {
			serveError(w, err)
		}

		ret := ""
		for _, v := range dir {

			stat := v.Sys().(*syscall.Stat_t)
			ret += fmt.Sprint(v.Name(), " ", strconv.FormatUint(uint64(stat.Mode), 8), "\n")
		}

		fmt.Println("GET: on dir")
		fmt.Println(ret)

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(ret))

		// signal writeHead we're done with io
		c <- nil
		return
	}

	_, err = io.Copy(w, f)
	if err != nil {
		serveError(w, err)
	}

	// signal writeHead we're done with io
	c <- nil
}

// write or create files
func servePut(w http.ResponseWriter, r *http.Request, userSpace sandbox.Store) {

	//no put requests for directories
	if r.URL.Path[len(r.URL.Path)-1] == '/' {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("use POST to create directory"))
		return
	}

	// ensure there is a proper mode header before
	// before costly io operations
	mode, err := getMode(r)
	if err != nil {
		serveError(w, err)
		return
	}

	requestPath := r.URL.Path

	f, err := userSpace.OpenFile(requestPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		serveError(w, err)
		return
	}
	defer f.Close()

	p := f.Name()

	i, err := io.Copy(f, r.Body)
	defer r.Body.Close()
	if err != nil {
		serveError(w, err)
	}

	//assert atime and mtime
	atime, err := strconv.ParseInt(r.Header.Get("Atime"), 10, 64)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("atime header could not be parsed"))
		return
	}
	mtime, err := strconv.ParseInt(r.Header.Get("Mtime"), 10, 64)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("mtime header could not be parsed"))
		return
	}

	// set atime and mtime according to request
	userSpace.Chtimes(requestPath, time.Unix(atime, 0), time.Unix(mtime, 0))

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("wrote " + strconv.Itoa(int(i)) + " bytes to " + p))
}

// exclusive creation
func servePost(w http.ResponseWriter, r *http.Request, userSpace sandbox.Store) {

	mode, err := getMode(r)
	if err != nil {
		serveError(w, err)
		return
	}

	requestPath := r.URL.Path

	if r.URL.Path[len(r.URL.Path)-1] == '/' {
		err = userSpace.MkDir(requestPath, mode)
		if err != nil {
			serveError(w, err)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("created directory " + requestPath))
		return
	}

	f, err := userSpace.OpenFile(requestPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
	if err != nil {
		serveError(w, err)
		return
	}

	w.WriteHeader(200)
	w.Write([]byte("created " + f.Name()))
}

// delete things
func serveDelete(w http.ResponseWriter, r *http.Request, userSpace sandbox.Store) {

	requestPath := r.URL.Path

	err := userSpace.Remove(requestPath)
	if err != nil {
		serveError(w, err)
		return
	}

	w.WriteHeader(200)
	w.Write([]byte("deleted " + requestPath))
}

//
// Simple handler
//

type h struct {
	store *sandbox.UserStore
}

func (h h) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Println("\n---")
	fmt.Println(r.Method, ": ", r.URL)
	fmt.Println(r.Header)

	user, ok := auth(r.Header.Get("Authorization"))
	if !ok {
		http.Error(w, "could not authenticate", http.StatusForbidden)
		return
	}

	userSpace, err := h.store.User(user)
	if err != nil {
		http.Error(w, "not authorized", http.StatusForbidden)
		return
	}

	// TODO move to concerning handlers
	metaQuery := r.URL.Query().Get("meta")
	if metaQuery != "" {
		//do the meta query
		//ie walk
	}

	switch r.Method {
	case http.MethodHead:
		//serve file info
		serveHead(w, r, userSpace)
	case http.MethodGet:
		//get file
		serveGet(w, r, userSpace)
	case http.MethodPut:
		//overwrite file
		servePut(w, r, userSpace)
	case http.MethodPost:
		//make new file
		servePost(w, r, userSpace)
	case http.MethodDelete:
		//delete file
		serveDelete(w, r, userSpace)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
	fmt.Println("\nheaders", w.Header())
	fmt.Println("sent----")
}

func main() {
	fmt.Println("server starting")

	us, err := sandbox.NewUserStore("store")
	if err != nil {
		panic(err)
	}

	fmt.Println("store created")

	http.ListenAndServe(":8080", h{
		store: us,
	})
}
