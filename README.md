# R3stFs
A remote filesystem made wit FUSE and a restful back end.

``` bash
$ go build ./client
$ go build ./server
$ echo "some text" > /var/ear7h/user/file.txt
$ server/server
$ client/client ./fs
```

__TODO__
* rename files
* comment code
* write tests
* refactor cache to be package
* make command line util

__DONE__
* read
* write
* rename
* delete