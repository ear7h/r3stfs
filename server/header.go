package server

import (
	"net/http"
	"fmt"
	"syscall"
	"strconv"
	"time"
	"os"
)

/*
HEAD /dir/file.txt

200
File-Mode: 0777
Last-Modified: Mon, 02 Jan 2006 15:04:05 MST //rfc 1123
Atime: 15324230 // Unix time
Mtime: 15324230 // Unix time
*/

// write file attributes to passed ResponseWriter's header
//
// writeHead will make one write to the errc channel
// a signal to the caller to proceed with io
// 		if it is not nil don't do io
// and a read
//		this is to hang until the caller is done with io before
//		asserting the old mtime and atime

// write header based on specific file
func writeHead(header *http.Header, localFile string) (error) {
	header.Set("Trailer", "File-Mode Last-Modified Mtime Atime")

	fi, err := os.Stat(localFile)
	if err != nil {
		fmt.Println("wh there was error")
		return err
	}
	// signal the caller it is okay to access files

	stat, ok := fi.Sys().(*syscall.Stat_t)
	if !ok {
		fmt.Println("wh there was error")
		return err
	}

	atime := stat.Atimespec.Sec
	mtime := stat.Mtimespec.Sec

	header.Set("File-Mode", strconv.FormatUint(uint64(stat.Mode), 8))
	header.Set("Last-Modified", fi.ModTime().Format(time.RFC1123))
	header.Set("Mtime", strconv.FormatInt(mtime, 10))
	header.Set("Atime", strconv.FormatInt(atime, 10))

	/* should be handled in specific handlers
	if !fi.IsDir() {
		header.Set("Content-Length", strconv.FormatInt(fi.Size(), 10))
	}
	*/

	fmt.Println("header written\n", header)

	return nil
}