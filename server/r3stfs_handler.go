package main

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

func (h *R3stFsHandler)	HandleGet(header http.Header, path string) (io.ReadCloser, error) {
	modeStr := header.Get("File-Mode")
	modeUint, err := strconv.ParseUint(modeStr, 8, 32)
	if err != nil {
		return nil, err
	}

	switch mode := os.FileMode(modeUint); mode {
	case 0: // file
		file, err := os.OpenFile(path, 0700, mode)
		if err != nil {
			return nil, err
		}

		return file, nil

	case os.ModeDir: // dir
		dir, err := ioutil.ReadDir(path)
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
