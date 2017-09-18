package gc

import (
	"time"
	"os"
)

var MOUNTPOINT string
var MOUNTED bool
var FILE_LOCKED map[string]bool
//this needs to be optimized


//clean cache after it becomes larger than 16 GB

func loop() {
	for MOUNTED {
		if needs() {
			do()
		}

		time.Sleep(1 * time.Minute)
	}
}

//check size of mount directory
func needs() bool {

	stat, err := os.Stat(MOUNTPOINT)
	if err != nil {

		return false
	}
	return stat.Size() > 16e+9
}

func do() {

}

func Lock(p string) {
	FILE_LOCKED[p] = true
}

func Unlock(p string) {
	delete(FILE_LOCKED, p)
}

func Start(mtpt string) {
	MOUNTPOINT = mtpt
	MOUNTED = true
	go loop()
}

func Stop() {
	MOUNTED = false
}