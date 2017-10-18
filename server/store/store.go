package store

import (
	"os"
	"path"
	"syscall"
	"time"
)

const STORAGE_DIR = "/var/ear7h/r3stfs"

func absPath(user, file string) string {
	return path.Join(STORAGE_DIR, "users", user, file)
}

func Stat(user, file string) (fi os.FileInfo, err error) {
	p := absPath(user, file)

	fi, err = os.Stat(p)
	return
}

func Open(user, file string) (*os.File, error) {
	return OpenFile(user, file, os.O_RDONLY, 0666)
}

func OpenFile(user, file string, flag int, perm os.FileMode) (*os.File, error) {
	p := absPath(user, file)

	return os.OpenFile(p, flag, perm)
}

func MkDir (user, file string, perm os.FileMode) error {
	p := absPath(user, file)

	return os.Mkdir(p, perm)
}

func Delete(user, file string) error {
	p := absPath(user, file)

	return os.Remove(p)
}

func Chtimes(user, file string, atime, mtime time.Time) {
	p := absPath(user, file)

	os.Chtimes(p, atime, mtime)
}