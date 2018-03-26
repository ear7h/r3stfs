package server

import (
	"net/http"
	"io"
	"strconv"
	"os"
	"io/ioutil"
	"bytes"
	"fmt"
	"encoding/json"
)

type R3stFsHandler struct {
}

func (h *R3stFsHandler) HandleGet(header http.Header, filename string) (io.ReadCloser, error) {
	var mode os.FileMode

	modeStr := header.Get("File-Mode")
	// if no file mode provided imply an existing file
	if modeStr == "" {
		stat, err := os.Stat(filename)
		if os.IsNotExist(err) {
			return nil, err
		}

		// use mode of exisring file
		mode = stat.Mode()
	} else {
		modeUint, err := strconv.ParseUint(modeStr, 8, 32)
		if err != nil {
			return nil, err
		}

		mode = os.FileMode(modeUint)
	}

	switch mode | os.ModeType {
	case 0: // file
		file, err := os.OpenFile(filename, os.O_RDONLY, mode)
		if err != nil {
			return nil, err
		}

		return file, nil

	case os.ModeDir: // dir
		dir, err := ioutil.ReadDir(filename)
		if err != nil {
			return nil, err
		}

		ret := bytes.NewBuffer(make([]byte, 20))
		encoder := json.NewEncoder(ret)

		data := make(map[string]os.FileMode)
		for _, node := range dir {
			data[node.Name()] = node.Mode()
		}

		err = encoder.Encode(data)
		if err != nil {
			return nil, err
		}

		return ioutil.NopCloser(ret), nil

	case os.ModeSymlink: // TODO: handle other modes
		fallthrough
	case os.ModeSocket:
		fallthrough
	case os.ModeNamedPipe:
		fallthrough
	default:
		str := fmt.Sprintf("mode %s not implemented", modeStr)
		return nil, NewNotImplementedError(str)
	}
}

func (h *R3stFsHandler) HandlePut(header http.Header, filename string, body io.Reader) (int, error) {
	if filename[len(filename)-1] == '/' {
		return 0, NewUserError("use POST to create directory")
	}

	modeStr := header.Get("File-Mode")
	modeUint, err := strconv.ParseUint(modeStr, 8, 32)
	if err != nil {
		return 0, WrapUserError(err)
	}

	switch mode := os.FileMode(modeUint); mode | os.ModeType {
	case 0: //file
		f, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
		if err != nil {
			return 0, err
		}

		num, err := io.Copy(f, body)
		if err != nil {
			return 0, err
		}
		f.Close()

		return int(num), nil

	case os.ModeDir:
		return 0, NewUserError("use POST to create directory")

	case os.ModeSymlink: // TODO: handle other modes
		fallthrough
	case os.ModeSocket:
		fallthrough
	case os.ModeNamedPipe:
		fallthrough
	default:
		str := fmt.Sprintf("mode %s not implemented", modeStr)
		return 0, NewNotImplementedError(str)
	}
}

// Exclusive create
func (h *R3stFsHandler) HandlePost(header http.Header, filename string, body io.Reader) (int, error) {
	modeStr := header.Get("File-Mode")
	modeUint, err := strconv.ParseUint(modeStr, 8, 32)
	if err != nil {
		return 0, WrapUserError(err)
	}

	switch mode := os.FileMode(modeUint); mode | os.ModeType {
	case 0: //file
		f, err := os.OpenFile(filename, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
		if err != nil {
			return 0, err
		}

		num, err := io.Copy(f, body)
		if err != nil {
			return 0, err
		}

		f.Close()

		return int(num), nil
	case os.ModeDir:
		err := os.Mkdir(filename, mode)
		if err != nil {
			return 0, err
		}
		return 0, nil

	case os.ModeSymlink: // TODO: handle other modes
		fallthrough
	case os.ModeSocket:
		fallthrough
	case os.ModeNamedPipe:
		fallthrough
	default:
		str := fmt.Sprintf("mode %s not implemented", modeStr)
		return 0, NewNotImplementedError(str)
	}
}

func (h *R3stFsHandler) HandleDelete(header http.Header, filename string) (error) {
	modeStr := header.Get("File-Mode")
	modeUint, err := strconv.ParseUint(modeStr, 8, 32)
	if err != nil {
		return WrapUserError(err)
	}

	switch mode := os.FileMode(modeUint); mode | os.ModeType {
	case 0: //file
		fallthrough
	case os.ModeDir:
		return os.Remove(filename)

	case os.ModeSymlink: // TODO: handle other modes
		fallthrough
	case os.ModeSocket:
		fallthrough
	case os.ModeNamedPipe:
		fallthrough
	default:
		str := fmt.Sprintf("mode %s not implemented", modeStr)
		return NewNotImplementedError(str)
	}
}

func (h *R3stFsHandler) HandleOptions(header http.Header, filename string) (io.ReadCloser, error) {
	var mode os.FileMode

	modeStr := header.Get("File-Mode")
	if modeStr == "" {
		// check if if file exists
		stat, err := os.Stat(filename)
		if err != nil {
			if os.IsNotExist(err) { // request querying for options on an empty path
				ret, err := jsonReader(map[string]string{
					"POST": "create new file or directory",
					"PUT":  "create and overwrite files",
				})

				return ioutil.NopCloser(ret), err
			}

			// all other errors
			return nil, WrapUserError(err)
		}

		// set the existing file's mode to the funcitons mode variable
		mode = stat.Mode()
	} else { // request is specifying a file mode
		modeUint, err := strconv.ParseUint(modeStr, 8, 32)
		if err != nil {
			return nil, WrapUserError(err)
		}

		mode = os.FileMode(modeUint)
		// check the user supplied mode is ok
		stat, err := os.Stat(filename)
		if err != nil {
			return nil, err
		}

		// if the user specified mode and the actual
		// file mode are not the same
		// return an error
		if (stat.Mode() ^ mode) | os.ModeType == 0 {
			return nil, NewUserError("specified mode does not match file mode")
		}
	}

	var value interface{}

	switch mode | os.ModeType {
	case 0: //file
		value = map[string]string{
			"GET":    "response body contains file",
			"POST":   "exclusively create new files",
			"PUT":    "create and overwrite files",
			"DELETE": "remove files",
		}
	case os.ModeDir:
		value = map[string]string{
			"GET":    "response body contains a json object with file keys and file mode values",
			"POST":   "create a directory",
			"PUT":    "not allowed",
			"DELETE": "remove directory",
		}

	case os.ModeSymlink: // TODO: handle other modes
		fallthrough
	case os.ModeSocket:
		fallthrough
	case os.ModeNamedPipe:
		fallthrough
	default:
		value = map[string]string{
			"GET":    "not implemented",
			"POST":   "not implemented",
			"PUT":    "not implemented",
			"DELETE": "not implemented",
		}
	}

	ret, err := jsonReader(value)

	return ioutil.NopCloser(ret), err

}

// helper function
func jsonReader(v interface{}) (io.Reader, error) {
	ret := bytes.NewBuffer(make([]byte, 20))
	encoder := json.NewEncoder(ret)

	err := encoder.Encode(v)

	return ret, err
}
