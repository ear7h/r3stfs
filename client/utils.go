package main

import (
	"net/http"
	"path"

	"r3stfs/client/remote"
)

type cacheStatus = int

const (
	CACHE_ENOENT cacheStatus = iota
	CACHE_GOOD
	CACHE_BAD
)

//TODO: implement MD5
func trustCache(name string) cacheStatus {
	resp, err := remote.Head(name)
	if err != nil {
		return CACHE_BAD
	}
	if resp.StatusCode == http.StatusNotFound {
		return CACHE_ENOENT
	}

	return CACHE_BAD
}

func localPath(name string) string {
	return path.Join(G_LOCAL_DIR, G_REMOTE_DOMAIN ,name)
}
