package tststan

import (
	"io/ioutil"
	"log"
	"os"
	"testing"

	"github.com/nats-io/go-nats-streaming"
	"github.com/stretchr/testify/assert"
)

const (
	clientId = "tststanClient"
	subject  = "tststanSubject"
)

func TestRun(t *testing.T) {
	assert := assert.New(t)
	var err error

	// Create temp directory.
	tmpDir, err := ioutil.TempDir("/tmp", "tststan")
	if err != nil {
		log.Panic(err)
	}
	defer os.RemoveAll(tmpDir)
	log.Printf("Temp directory created: %s\n", tmpDir)

	// Use the temp directory as host data path.
	opts := &Options{
		FileStore:    true,
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
	log.Printf("The first nats streaming server is up, nats url: %+q.\n", res1.NatsURL())
	log.Printf("%#v\n", res1.Options)

	// Connect to the first server.
	var client1 stan.Conn
	{
		client1, err = res1.StanClient(clientId)
		assert.NoError(err)
		defer client1.Close()
	}
	log.Printf("Connected to the first nats streaming server.\n")

	// Subscribe.
	handler1c := make(chan struct{})
	{
		_, err := client1.Subscribe(subject, func(m *stan.Msg) {
			handler1c <- struct{}{}
		})
		assert.NoError(err)
	}
	log.Printf("Subscribed to %+q.\n", subject)

	// Publish.
	{
		err := client1.Publish(subject, []byte("good"))
		assert.NoError(err)
		<-handler1c
	}
	log.Printf("Publishd to %+q and handled.\n", subject)

	// Stop the first server/client.
	res1.Close()
	client1.Close()
	log.Printf("The first nats streaming server is down.\n")

	// Run the second server.
	var res2 *Resource
	{
		res2, err = Run(opts)
		assert.NoError(err)
		defer res2.Close()
	}
	log.Printf("The second nats streaming server is up, nats url: %+q.\n", res2.NatsURL())
	log.Printf("%#v\n", res2.Options)

	// Connect to the second server using raw nats connection.
	var client2 stan.Conn
	{
		nc, err := res2.NatsClient()
		assert.NoError(err)
		defer nc.Close()

		client2, err = stan.Connect(res2.Options.ClusterId, clientId, stan.NatsConn(nc))
		assert.NoError(err)
		defer client2.Close()
	}
	log.Printf("Connected to the second nats streaming server.\n")

	// Subscribe.
	handler2c := make(chan struct{})
	{
		_, err := client2.Subscribe(
			subject,
			func(m *stan.Msg) {
				handler2c <- struct{}{}
			},
			stan.DeliverAllAvailable(), // Should get the previous msg.
		)
		assert.NoError(err)
	}
	log.Printf("Subscribed to %+q.\n", subject)

	// Recv previous msg.
	<-handler2c
	log.Printf("Received previous message.\n")
}
