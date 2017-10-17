package remote

import (
	"io"
	"net/http"
	"encoding/base64"
	"fmt"
	"os"
	"strconv"
	"path"
)

var g_host, g_user, g_token string
var g_httpClient http.Client

func writeHeader(r *http.Request) {
	b64Username := base64.StdEncoding.EncodeToString([]byte(g_user))

	r.Header["Authorization"] = []string{"basic " + b64Username}

	return
}

//get expecting a file
func Get(urlPath string) (*http.Response, error){

	u := fmt.Sprint("http://", g_host, "/", urlPath)

	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}

	writeHeader(req)

	return g_httpClient.Do(req)

}

func Post(urlPath string, body io.Reader) (*http.Response, error){
	u := fmt.Sprint("http://", g_host, "/", urlPath)

	req, err := http.NewRequest(http.MethodPost, u, body)
	if err != nil {
		return nil, err
	}

	writeHeader(req)

	return g_httpClient.Do(req)
}

func Put(urlPath string, body io.Reader, mode os.FileMode) (*http.Response, error){
	p := path.Join("/",urlPath)
	u := fmt.Sprintf("http://%s%s", g_host, p)

	req, err := http.NewRequest(http.MethodPut, u, body)
	if err != nil {
		return nil, err
	}

	writeHeader(req)
	req.Header.Set("File-Mode", strconv.FormatInt(int64(mode), 8))

	return g_httpClient.Do(req)
}

func Head(urlPath string) (*http.Response, error) {
	u := fmt.Sprint("http://", g_host, "/", urlPath)

	req, err := http.NewRequest(http.MethodHead, u, nil)
	if err != nil {
		return nil, err
	}

	writeHeader(req)

	return g_httpClient.Do(req)
}

func Delete(urlPath string) (*http.Response, error){
	u := fmt.Sprint("http://", g_host, "/", urlPath)

	req, err := http.NewRequest(http.MethodDelete, u, nil)
	if err != nil {
		return nil, err
	}

	writeHeader(req)

	return g_httpClient.Do(req)
}

//wrapper for head
//TODO: optimize as own request
func Sum(urlPath string) (string, error) {
	resp, err := Head(urlPath)
	if err != nil {
		return "", err
	}

	return resp.Header.Get("MD5"), nil
}

func Login(host, user, pass string) {
	g_host = host
	g_user = user
	g_token = pass

	//get token

}