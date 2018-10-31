package tstsvc

import (
	"log"
	"net"

	"github.com/ory/dockertest"
)

var (
	defaultPool *dockertest.Pool
)

func init() {
	var err error
	defaultPool, err = dockertest.NewPool("")
	if err != nil {
		log.Fatal(err)
	}
}

// DefaultPool returns the default dockertest Pool.
func DefaultPool() *dockertest.Pool {
	return defaultPool
}

// FreePort returns a free tcp port number ready to use.
// See: https://stackoverflow.com/a/43425461/157235
func FreePort() uint16 {
	l, err := net.Listen("tcp4", ":0")
	if err != nil {
		panic(err)
	}
	defer l.Close()
	return uint16(l.Addr().(*net.TCPAddr).Port)
}
