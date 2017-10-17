package main

import (
	"path"
)


//TODO: implement MD5
func cacheOK(name string) bool {
	return true
}

func localPath(name string) string {
	return path.Join(G_LOCAL_DIR, G_REMOTE_DOMAIN, name)
}
