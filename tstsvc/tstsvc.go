package tstsvc

import (
	"log"
	"math/rand"
	"time"

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

	rand.Seed(time.Now().UnixNano())
}

// DefaultPool returns the default dockertest Pool.
func DefaultPool() *dockertest.Pool {
	return defaultPool
}

// RandPort returns a random port number.
func RandPort() uint16 {
	return 50000 + uint16(rand.Int()%14983)
}
