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
* comment code
* write tests
* create logger
* correct renames

__DONE__
* read
* write