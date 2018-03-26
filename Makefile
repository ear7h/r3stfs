toplevel_sources = $(wildcard ./*.go)

top:
	echo "please specify server or client"

topsrc:
	echo $(toplevel_sources)

server: $(toplevel_sources) ./server ./cmd/r3stfs-server/
	go build github.com/ear7h/r3stfs/cmd/r3stfs-server
	$GOPATH/github.com/ear7h/r3stfs/cmd/r3stfs-server/r3stfs-server