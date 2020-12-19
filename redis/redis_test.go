package tstredis

import (
	"context"
	"io/ioutil"
	"log"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRun(t *testing.T) {
	assert := assert.New(t)
	var err error
	ctx := context.Background()

	// Create temp directory.
	tmpDir, err := ioutil.TempDir("/tmp", "tstredis")
	if err != nil {
		log.Panic(err)
	}
	defer os.RemoveAll(tmpDir)
	log.Printf("Temp directory created: %s\n", tmpDir)

	// Use the temp directory as host data path.
	opts := &Options{
		HostDataPath: tmpDir,
	}
	log.Printf("%#v\n", *opts)

	// Run the first server.
	var res1 *Resource
	{
		res1, err = Run(opts)
		assert.NoError(err)
		defer res1.Close()
	}
	log.Printf("The first redis server is up, addr: %+q.\n", res1.Addr())
	log.Printf("%#v\n", res1.Options)

	// Creates the first client.
	var client1 = res1.Client()
	defer client1.Close()
	log.Printf("The first client created\n")

	key := "keykey"
	value := "valval"

	// Sets a key/value.
	assert.NoError(client1.Set(ctx, key, value, 0).Err())
	log.Printf("A key/value pair set\n")

	// Save to disk.
	assert.NoError(client1.Save(ctx).Err())
	log.Printf("Saved to disk.\n")

	// Close the first server/client.
	res1.Close()
	client1.Close()
	log.Printf("The first redis server is down.\n")

	// Run the second server.
	var res2 *Resource
	{
		res2, err = Run(opts)
		assert.NoError(err)
		defer res2.Close()
	}
	log.Printf("The second redis server is up, addr: %+q.\n", res2.Addr())
	log.Printf("%#v\n", res2.Options)

	// Creates the second client.
	var client2 = res2.Client()
	defer client2.Close()
	log.Printf("The second client created\n")

	// Get key and check.
	{
		v, err := client2.Get(ctx, key).Result()
		assert.NoError(err)
		assert.Equal(value, v)
	}
}
