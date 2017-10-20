/*
This package is an abstraction for os file functions which
sandboxes operations to a storage directory
*/

package sandbox

import (
	"os"
	"path"
	"path/filepath"
	"time"
	"fmt"
	"syscall"
)

type Store struct {
	root string
}

func (s Store) Abs(file string) string {
	fmt.Println("Abs: ", path.Join(s.root, file))
	return path.Join(s.root, file)
}

func (s Store) Stat(file string) (fi os.FileInfo, err error) {
	p := s.Abs(file)

	fi, err = os.Stat(p)
	return
}

func (s Store) Open(file string) (*os.File, error) {
	return s.OpenFile(file, os.O_RDONLY, 0666)
}

func (s Store) OpenFile(file string, flag int, perm os.FileMode) (*os.File, error) {
	p := s.Abs(file)

	return os.OpenFile(p, flag, perm)
}

func (s Store) MkDir(file string, perm os.FileMode) error {
	p := s.Abs(file)

	return os.Mkdir(p, perm)
}

func (s Store) MkDirAll(file string, perm os.FileMode) error {
	p := s.Abs(file)

	return os.MkdirAll(p, perm)
}

func (s Store) Remove(file string) error {
	p := s.Abs(file)

	return os.Remove(p)
}

func (s Store) RemoveAll(file string) error {
	p := s.Abs(file)

	return os.RemoveAll(p)
}

func (s Store) Rename(old, new string) error {
	pOld := s.Abs(old)
	pNew := s.Abs(new)

	return os.Rename(pOld, pNew)
}

func (s Store) Chtimes(file string, atime, mtime time.Time) {
	p := s.Abs(file)

	os.Chtimes(p, atime, mtime)
}

func (s Store) Access(file string, mode uint32) error {
	p := s.Abs(file)

	return syscall.Access(p, mode)
}

func (s Store) SelfDestruct() {
	os.Remove(string(s.root))
}

// psuedo constructor
func NewStore(root string) (s Store, err error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return
	}

	err = os.MkdirAll(root, 0700)
	if err != nil {
		return
	}

	s = Store{abs}
	return
}


// UserStore
type UserStore struct {
	root   string
}

func (us *UserStore) User(name string) (s Store, err error) {
	userRoot := path.Join(us.root, name)
	fmt.Println("looking in ", userRoot)

	//return error if user does not have file
	if _, err = os.Stat(userRoot); err != nil {
		err = fmt.Errorf("user %s does not exist", name)
		fmt.Println(err)
	} else {
		s = Store{
			root: userRoot,
		}
	}

	return
}

func (us *UserStore) SelfDestruct() {
	os.RemoveAll(us.root)
}

// psuedo constructor
func NewUserStore(root string) (us *UserStore, err error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return
	}

	err = os.MkdirAll(root, 0700)
	if err != nil {
		return
	}

	us = &UserStore{abs}
	return
}
