package tstsvc

import (
	"log"

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
