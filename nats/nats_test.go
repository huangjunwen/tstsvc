package tstnats

import (
	"log"
	"testing"
	"time"

	nats "github.com/nats-io/go-nats"
	"github.com/stretchr/testify/assert"

	"github.com/huangjunwen/tstsvc"
)

const (
	subject = "tstnatsSubject"
)

func TestRun(t *testing.T) {
	assert := assert.New(t)
	var err error

	opts := &Options{
		HostPort:        tstsvc.FreePort(),
		HostMonPort:     tstsvc.FreePort(),
		HostClusterPort: tstsvc.FreePort(),
	}
	log.Printf("%#v\n", *opts)

	// Run the first server.
	var res1 *Resource
	{
		res1, err = Run(opts)
		assert.NoError(err)
		defer res1.Close()
	}
	log.Printf("The first nats server is up, nats url: %+q.\n", res1.NatsURL())
	log.Printf("%#v\n", res1.Options)

	// Make the connection.
	var nc *nats.Conn
	{
		nc, err = res1.NatsClient(
			nats.MaxReconnects(-1), // Never give up reconnect.
			nats.ReconnectWait(100*time.Millisecond),
		)
		assert.NoError(err)
		defer nc.Close()
	}
	log.Printf("Connected to the first nats server.\n")

	// Subscribe.
	handlerc := make(chan struct{})
	{
		_, err := nc.Subscribe(subject, func(m *nats.Msg) {
			handlerc <- struct{}{}
		})
		assert.NoError(err)
	}
	log.Printf("Subscribed to %+q.\n", subject)

	// Publish.
	{
		err := nc.Publish(subject, []byte("good"))
		assert.NoError(err)
		<-handlerc
	}
	log.Printf("Publishd to %+q and handled.\n", subject)

	// Stop the first server.
	res1.Close()
	log.Printf("The first nats server is down.\n")

	// Run the second server.
	var res2 *Resource
	{
		res2, err = Run(opts)
		assert.NoError(err)
		defer res2.Close()
	}
	log.Printf("The second nats server is up, nats url: %+q.\n", res2.NatsURL())
	log.Printf("%#v\n", res2.Options)

	// Wait a while.
	time.Sleep(time.Second)

	// Publish again.
	{
		err := nc.Publish(subject, []byte("good"))
		assert.NoError(err)
		<-handlerc
	}
	log.Printf("Publishd again to %+q and handled.\n", subject)
}
