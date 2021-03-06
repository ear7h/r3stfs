package server

import (
	"net/http"
	"io"
	"strings"
	"path"
	"strconv"
	"os"
	"io/ioutil"
	"fmt"
	"log"
)

func ServeFs(addr, basepath, dirroot string, handler FsHandler) error {
	_, err := os.Stat(dirroot)
	if err != nil {
		if os.IsNotExist(err) {
			err := os.Mkdir(dirroot, 0700)
			if err != nil {
				panic(err)
			}

			fmt.Printf("root %s created", dirroot)
		} else {
			panic(err)
		}
	} else {
		fmt.Printf("root %s already exists", dirroot)
	}

	h := &fsHandlerWrapper{
		FsHandler: handler,
		basepath:  basepath,
		dirroot:   dirroot,
	}

	return http.ListenAndServe(addr, h)
}

type fsHandlerWrapper struct {
	FsHandler
	basepath, dirroot string
}

func stringReadCloser(str string) io.ReadCloser {
	return ioutil.NopCloser(strings.NewReader(str))
}

func (h *fsHandlerWrapper) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(h.basepath, r.URL.Path) {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}

	filename := r.URL.Path[len(h.basepath):]
	filename = path.Join(h.basepath, filename)

	var res io.ReadCloser
	var err error

	defer func(rc io.ReadCloser) {
		if res != nil {
			res.Close()
		}
	}(res)

	switch r.Method {
	case http.MethodGet:
		res, err = h.HandleGet(r.Header, filename)
	case http.MethodPut:
		var num int
		num, err = h.HandlePut(r.Header, filename, r.Body)
		defer r.Body.Close()
		res = stringReadCloser(strconv.Itoa(num))
	case http.MethodPost:
		var num int
		num, err = h.HandlePost(r.Header, filename, r.Body)
		defer r.Body.Close()
		res = stringReadCloser(strconv.Itoa(num))
	case http.MethodDelete:
		err = h.HandleDelete(r.Header, filename)
		res = stringReadCloser("delete")
	case http.MethodOptions:
		res, err = h.HandleOptions(r.Header, filename)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err != nil {
		switch {
		case os.IsNotExist(err):
			// 404
			s := fmt.Sprintf("%s not found", filename)
			http.Error(w, s, http.StatusNotFound)
		case os.IsPermission(err):
			// forbidden
			s := fmt.Sprintf("%s access forbidden", filename)
			http.Error(w, s, http.StatusForbidden)
		case os.IsExist(err):
			// not allowed
			s := fmt.Sprintf("%s already exists", filename)
			http.Error(w, s, http.StatusConflict)
		case os.IsTimeout(err):
			// timeout
			s := fmt.Sprintf("%s i/o timed out", filename)
			http.Error(w, s, http.StatusInternalServerError)
		case IsNotImplemented(err):
			s := fmt.Sprintf("%s method %s not implemented", filename, r.Method)
			http.Error(w, s, http.StatusNotImplemented)
		case IsUser(err):
			s := fmt.Sprintf("%s %s", filename, err.Error())
			http.Error(w, s, http.StatusBadRequest)
		default:
			// internal server error
			log.Printf("unexpected error %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
		}
		return
	}

	// header should be written after http method handling to ensure
	// the requested file has already been made
	headerErr := make(chan error)
	go func() {
		header := w.Header()
		headerErr <- writeHead(&header, filename)
	}()

	// copy response and check for errors
	_, err = io.Copy(w, res)
	if err != nil {
		http.Error(w, "error writing response", http.StatusInternalServerError)
	}

	// check header error
	err = <-headerErr
	if err != nil {
		http.Error(w, "error writing header", http.StatusInternalServerError)
	}
}
