package main

import (
	"fmt"
	"net/http"
	"os"
	"path"
	"r3stfs/sandbox"
	"testing"
	"io"
	"strings"
	"time"
	"strconv"
)

const TEST_STORE = "test_store"
const TEST_USER = "test_user"
const TEST_USER64 = "dGVzdF91c2Vy"


func generateFiles(s *sandbox.UserStore) {
	err := os.Mkdir(path.Join(TEST_STORE, TEST_USER), 0777)
	if err != nil {
		panic(err)
	}

	t, err := s.User(TEST_USER)
	if err != nil {
		panic(err)
	}

	f, err := t.OpenFile("hello.txt", os.O_RDWR | os.O_CREATE, 0777)
	if err != nil {
		panic(err)
	}
	fmt.Fprint(f, "asdasd")
	f.Close()

	f, err = t.OpenFile("file2.go", os.O_RDWR | os.O_CREATE, 0777)
	if err != nil {
		panic(err)
	}
	fmt.Fprint(f, "package main\nmain(){}")
	f.Close()
}

func startServer() *sandbox.UserStore {
	fmt.Println("server starting")
	us, err := sandbox.NewUserStore(TEST_STORE)
	if err != nil {
		panic(err)
	}

	fmt.Println("store created")

	go http.ListenAndServe(":8080", h{
		store: us,
	})

	return us
}

func TestGetFile(t *testing.T) {
	us := startServer()
	generateFiles(us)

	//write headers
	req, err := http.NewRequest(http.MethodGet, "http://localhost:8080/hello.txt", nil)
	if err != nil {
		panic(err)
	}

	req.Header.Set("Authorization", "basic " + TEST_USER64)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}

	io.Copy(os.Stdout, res.Body)
	res.Body.Close()

	us.SelfDestruct()
}

func TestGetDir(t *testing.T) {
	us := startServer()
	generateFiles(us)

	//write headers
	req, err := http.NewRequest(http.MethodGet, "http://localhost:8080/", nil)
	if err != nil {
		panic(err)
	}

	req.Header.Set("Authorization", "basic " + TEST_USER64)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}

	io.Copy(os.Stdout, res.Body)
	res.Body.Close()

	us.SelfDestruct()
}

func TestHead(t *testing.T) {
	us := startServer()
	generateFiles(us)

	//write headers
	req, err := http.NewRequest(http.MethodHead, "http://localhost:8080/hello.txt", nil)
	if err != nil {
		panic(err)
	}
	req.Header.Set("Authorization", "basic " + TEST_USER64)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}

	fmt.Println(res.Header)

	us.SelfDestruct()
}

func TestPut(t *testing.T) {
	us := startServer()
	generateFiles(us)


	//write request
	req, err := http.NewRequest(http.MethodPut, "http://localhost:8080/myfile.txt", strings.NewReader("a new file\n"))
	if err != nil {
		panic(err)
	}
	req.Header.Set("Authorization", "basic " + TEST_USER64)
	req.Header.Set("File-Mode", "0777")
	req.Header.Set("Atime", strconv.FormatInt(int64(time.Now().Second()), 10))
	req.Header.Set("Mtime", strconv.FormatInt(int64(time.Now().Second()), 10))

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}

	io.Copy(os.Stdout, res.Body)
	res.Body.Close()

	//write headers
	req, err = http.NewRequest(http.MethodGet, "http://localhost:8080/myfile.txt", nil)
	if err != nil {
		panic(err)
	}

	req.Header.Set("Authorization", "basic " + TEST_USER64)

	res, err = http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}

	io.Copy(os.Stdout, res.Body)
	res.Body.Close()


	us.SelfDestruct()
}

