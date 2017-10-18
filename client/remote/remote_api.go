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
)

type Client struct {
	host, user, token string
	b64Username       string
	http              http.Client
}

//get expecting a file
func (c *Client) Get(urlPath string) (*http.Response, error) {

	u := fmt.Sprint("http://", c.host, "/", urlPath)

	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "basic "+c.b64Username)

	return c.http.Do(req)

}

func (c *Client) Post(urlPath string, body io.Reader) (*http.Response, error) {
	u := fmt.Sprint("http://", c.host, "/", urlPath)

	req, err := http.NewRequest(http.MethodPost, u, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "basic "+c.b64Username)

	return c.http.Do(req)
}

func (c *Client) Put(urlPath string, file *os.File) (*http.Response, error) {
	p := path.Join("/", urlPath)
	u := fmt.Sprintf("http://%s%s", c.host, p)

	fi, err := file.Stat()
	if err == nil {
		return nil, err
	}

	mode := fi.Mode()
	stat := fi.Sys().(*syscall.Stat_t)
	atime := stat.Mtimespec.Sec
	mtime := stat.Mtimespec.Sec

	req, err := http.NewRequest(http.MethodPut, u, file)
	if err != nil {
		return nil, err
	}

	req.Header.Set("File-Mode", strconv.FormatInt(int64(mode), 8))
	req.Header.Set("Atime", strconv.FormatInt(atime, 10))
	req.Header.Set("Mtime", strconv.FormatInt(mtime, 10))

	req.Header.Set("Authorization", "basic "+c.b64Username)

	return c.http.Do(req)
}

func (c *Client) Head(urlPath string) (*http.Response, error) {
	u := fmt.Sprint("http://", c.host, "/", urlPath)

	req, err := http.NewRequest(http.MethodHead, u, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "basic "+c.b64Username)

	return c.http.Do(req)
}

func (c *Client) Delete(urlPath string) (*http.Response, error) {
	u := fmt.Sprint("http://", c.host, "/", urlPath)

	req, err := http.NewRequest(http.MethodDelete, u, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "basic "+c.b64Username)

	return c.http.Do(req)
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
