package main

import (
	"path"
	"r3stfs/client/remote"
	"github.com/0xAX/notificator"
	"time"
	"strings"
	"os"
)


//TODO: implement MD5
func cacheOK(name string) bool {

	resp, err := remote.Head(name)
	if err != nil {
		notify := notificator.New(notificator.Options{
			AppName: "r3stfs",
		})

		notify.Push("Remote Error", "Can't reach cache", "", notificator.UR_NORMAL)
		return true
	}

	remoteModTime, err := time.Parse(time.RFC1123, strings.Join(resp.Header["Mode"], " "))

	stat, err := os.Stat(localPath(name))
	if err != nil {
		return true
	}

	// cache is outdated
	if remoteModTime.After(stat.ModTime()) {
		return false
	}

	return true
}

func localPath(name string) string {
	return path.Join(G_LOCAL_DIR, G_REMOTE_DOMAIN, name)
}
