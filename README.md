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
* groups in server
    * file locks (querystring in head call)
* command line args
* implement security
* make good tests
* cache garbage collection

__DONE__
* read
* write
* rename
* delete
* refactor to remove globals