package main

import (
	"net/http"
	"io"
)

// FsHandler is an interface which defines a serving abstraction over the filesystem
// the file names at this point are assumed to be safe
type FsHandler interface {
	// Read files
	HandleGet(header http.Header, filename string) (io.ReadCloser, error)
	// Create or write files
	HandlePut(header http.Header, filename string, body io.Reader) (int, error)
	// Exclusive create
	HandlePost(header http.Header, filename string, body io.Reader) (int, error)
	// Delete files
	HandleDelete(header http.Header, filename string) (error)
	// Check available methods for file
	HandleOptions(header http.Header, filename string) (io.ReadCloser, error)
}
