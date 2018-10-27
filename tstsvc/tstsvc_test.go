package tstsvc

import (
	"log"
	"testing"
)

func TestRandPort(t *testing.T) {
	for i := 0; i < 10; i++ {
		log.Println(FreePort())
	}
}
