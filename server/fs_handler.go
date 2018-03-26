package main

import (
	"net/http"
	"io"
)

// FsHandler is an interface which defines a serving abstraction over the filesystem
// the file names at this point are assumed to be safe
type FsHandler interface {
	HandleGet(header http.Header, filename string) (io.ReadCloser, error)
	HandlePut(header http.Header, filename string, body io.ReadCloser) (int, error)
	HandlePost(header http.Header, filename string, body io.ReadCloser) (int, error)
	HandleDelete(header http.Header, filename string) (error)
	HandleOptions(header http.Header, filename string) (io.ReadCloser, error)
}
