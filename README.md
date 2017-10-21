# R3stFs
A remote filesystem made wit FUSE and a restful back end.

``` bash
$ go build ./client
$ go build ./server
$ echo "some text" > ./server/store/user/file.txt
$ server/server &
$ client/client ./fs
```

__TODO__
* comment code
* write tests
* implement locks in server
* implement groups in server
* make command line util
* implement security
* make good tests

__DONE__
* read
* write
* rename
* delete
* refactor to remove globals