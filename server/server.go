package main

import (
	"crypto/md5"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"os"
	"r3stfs/server/store"
	"strconv"
	"strings"
	"time"
)

type FsRequest struct {
	user, file string
}

//called only by top level handlers
func (fr *FsRequest) error(w http.ResponseWriter, err error) {
	fmt.Println("error\n", err)

	if os.IsNotExist(err) {
		http.Error(w, "enoent", http.StatusNotFound)
	} else {
		http.Error(w, "oops", http.StatusInternalServerError)
	}

	return
}

//wrapper for writeHead
func (fr *FsRequest) serveHead(w http.ResponseWriter, r *http.Request) {
	c := make(chan error, 1)

	fr.writeHead(w, c)

	err := <-c
	if err != nil {
		fr.error(w, err)
		return
	}

	w.Write([]byte{})
}

/*
HEAD /dir/file.txt

200
File-Mode: 0777
Last-Modified: Mon, 02 Jan 2006 15:04:05 MST //rfc 1123
MD5: 1231asdasd //base64
Content-Length: 1000 //in bytes
*/

//make and write header if possible else return nil
//this is a utility function which should be run concurrently
func (fr *FsRequest) writeHead(w http.ResponseWriter, errc chan error) {
	defer close(errc)

	f, err := store.Open(fr.user, fr.file)
	if err != nil {
		errc <- err
		return
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		errc <- err
		return

	}


	s := md5.New()
	io.Copy(s, f)
	md5str := base64.StdEncoding.EncodeToString(s.Sum(nil))
	mode := strconv.FormatInt(int64(stat.Mode() & 0777), 8)

	w.Header()["Name"] = []string{f.Name()}
	w.Header()["File-Mode"] = []string{mode}
	w.Header()["Is-Dir"] = []string{strconv.FormatBool(stat.Mode().IsDir())}
	w.Header()["Last-Modified"] = []string{stat.ModTime().Format(time.RFC1123)}
	w.Header()["MD5"] = []string{md5str}
	if !stat.IsDir() {
		w.Header()["Content-Length"] = []string{strconv.FormatInt(stat.Size(), 10)}
	}

	errc <- nil
}

//gets file mode from request header
func (fr *FsRequest) getMode(r *http.Request) (os.FileMode, error) {
	fmt.Println("header", r.Header)

	i, err := strconv.ParseInt(r.Header.Get("File-Mode"), 8, 32)
	if err != nil {
		return 0, err
	}

	fmt.Println("mode ", os.FileMode(int32(i)))

	return os.FileMode(int32(i)), nil
}

func (fr *FsRequest) ServeGet(w http.ResponseWriter, r *http.Request) {

	c := make(chan error, 1)

	go fr.writeHead(w, c)

	f, err := store.Open(fr.user, fr.file)
	if err != nil {
		fr.error(w, err)
		return
	}
	defer f.Close()


	stat, err := f.Stat()
	if err != nil {
		fr.error(w, err)
		return
	}

	if stat.IsDir() {
		dir, err := f.Readdir(0)
		if err != nil {
			fr.error(w, err)
		}

		ret := ""
		for _, v := range dir {
			ret += fmt.Sprint(v.Name(), " ",strconv.FormatInt(int64(v.Mode()), 8) , "\n")
		}

		err = <-c
		if err != nil {
			fr.error(w, err)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(ret))
		return
	}

	err = <-c
	//check proper errors
	if err != nil {
		fr.error(w, err)
		return
	}

	io.Copy(w, f)
}

func (fr *FsRequest) ServePut(w http.ResponseWriter, r *http.Request) {

	mode, err := fr.getMode(r)
	if err != nil {
		fr.error(w, err)
		return
	}

	if r.URL.Path[len(r.URL.Path) - 1] == '/' {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("use POST to create directory"))
		return
	}

	f, err := store.OpenFile(fr.user, fr.file, os.O_CREATE|os.O_WRONLY, mode)
	if err != nil {
		fr.error(w, err)
		return
	}

	p := f.Name()

	i, err := io.Copy(f, r.Body)
	if err != nil {
		fr.error(w, err)
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("wrote " + strconv.Itoa(int(i)) + " bytes to " + p))
}

func (fr *FsRequest) ServePost(w http.ResponseWriter, r *http.Request) {

	mode, err := fr.getMode(r)
	if err != nil {
		fr.error(w, err)
		return
	}

	if r.URL.Path[len(r.URL.Path) - 1] == '/' {
		err = store.MkDir(fr.user, fr.file, mode)
		if err != nil {
			fr.error(w, err)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("created directory " + fr.file))
		return
	}

	f, err := store.OpenFile(fr.user, fr.file, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
	if err != nil {
		fr.error(w, err)
		return
	}

	w.WriteHeader(200)
	w.Write([]byte("created " + f.Name()))
}

func (fr *FsRequest) ServeDelete(w http.ResponseWriter, r *http.Request) {

	err := store.Delete(fr.user, fr.file)
	if err != nil {
		fr.error(w, err)
		return
	}

	w.WriteHeader(200)
	w.Write([]byte("deleted " + fr.file))
}

type handler struct{}

func (h handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Println(r.Method, ": ", r.URL)

	user, ok := h.auth(r.Header.Get("Authorization"))
	if !ok {
		http.Error(w, "could not authenticate", http.StatusForbidden)
		return
	}

	fr := FsRequest{
		user: user,
		file: r.URL.Path,
	}

	metaQuery := r.URL.Query().Get("meta")
	if metaQuery != "" {
		//do the meta query
		//ie walk
	}

	switch r.Method {
	case http.MethodHead:
		//serve file info
		fr.serveHead(w, r)
	case http.MethodGet:
		//get file
		fr.ServeGet(w, r)
	case http.MethodPut:
		//overwrite file
		fr.ServePut(w, r)
	case http.MethodPost:
		//make new file
		fr.ServePost(w, r)
	case http.MethodDelete:
		//delete file
		fr.ServeDelete(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h handler) auth(header string) (string, bool) {
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

func main() {
	h := handler{}

	http.ListenAndServe(":8080", h)
}
