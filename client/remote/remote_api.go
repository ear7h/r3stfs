// Copyright 2017 Julio. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

package remote

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strconv"
	"syscall"
	"time"
	"r3stfs/client/log"
)

type Client struct {
	host, user, token string
	b64Username       string
	http              http.Client
}

//get expecting a file
func (c *Client) Get(urlPath string) (*http.Response, error) {

	u := fmt.Sprint("http://", c.host, "/", urlPath)

	req, err := newRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "basic "+c.b64Username)

	return c.http.Do(req)

}

func (c *Client) Post(urlPath string, body io.Reader) (*http.Response, error) {
	u := fmt.Sprint("http://", c.host, "/", urlPath)

	req, err := newRequest(http.MethodPost, u, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "basic "+c.b64Username)

	return c.http.Do(req)
}

func (c *Client) Put(urlPath string, file *os.File) (res *http.Response, err error) {
	log.Func(urlPath, file.Name())
	defer func() {
		log.Return(res, err)
	}()

	p := path.Join("/", urlPath)
	u := fmt.Sprintf("http://%s%s", c.host, p)

	fi, err := file.Stat()
	if err != nil {
		return nil, err
	}

	mode := fi.Mode()
	stat := fi.Sys().(*syscall.Stat_t)
	atime := stat.Mtimespec.Sec
	mtime := stat.Mtimespec.Sec

	req, err := newRequest(http.MethodPut, u, file)
	if err != nil {
		return nil, err
	}

	req.Header.Set("File-Mode", strconv.FormatInt(int64(mode), 8))
	req.Header.Set("Atime", strconv.FormatInt(atime, 10))
	req.Header.Set("Mtime", strconv.FormatInt(mtime, 10))

	req.Header.Set("Authorization", "basic "+c.b64Username)

	res, err = c.http.Do(req)
	return
}

func (c *Client) Head(urlPath string) (*http.Response, error) {
	u := fmt.Sprint("http://", c.host, "/", urlPath)

	req, err := newRequest(http.MethodHead, u, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "basic "+c.b64Username)

	return c.http.Do(req)
}

func (c *Client) Delete(urlPath string) (*http.Response, error) {
	u := fmt.Sprint("http://", c.host, "/", urlPath)

	req, err := newRequest(http.MethodDelete, u, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "basic "+c.b64Username)

	return c.http.Do(req)
}

func newRequest(method, url string, body io.Reader) (req *http.Request, err error) {
	req, err = http.NewRequest(method, url, body)
	if err != nil {
		return
	}

	req.Header.Set("User-Agent", "ear7h-FUSE-client")
	return
}

//actual login, returns token
func login(host, user, pass string) string {
	return ""
}

func Login(host, user, pass string) *Client {
	return &Client{
		host:        host,
		user:        user,
		token:       login(host, user, pass),
		b64Username: base64.StdEncoding.EncodeToString([]byte(user)),
		http: http.Client{
			Timeout: 10 * time.Second,
		},
	}
}
