package sandbox

import (
	"testing"
	"math/rand"
)

func TestStore(t *testing.T) {
	s, err := NewStore("store_test_dir")
	defer s.SelfDestruct()
	if err != nil {
		panic(err)
	}
}

func TestUserStore(t *testing.T) {
	//make user Store
	uStore, err := NewUserStore("test")
	if err != nil {
		panic(err)
	}

	for i := 0; i < 100; i++ {
		byt := make([]byte, 10)
		rand.Read(byt)
		uStore.NewUser(string(byt))
	}
}