package tststan

import (
	"fmt"
	"time"

	nats "github.com/nats-io/nats.go"
	stan "github.com/nats-io/stan.go"
	"github.com/ory/dockertest"
	dc "github.com/ory/dockertest/docker"

	"github.com/huangjunwen/tstsvc"
)

var (
	// Docker repository.
	Repository = "nats-streaming"

	// Default tag.
	DefaultTag = "0.15.1-linux"

	// Default cluster id.
	DefaultClusterId = "tststan"

	// Default container expire time.
	DefaultExpire uint = 120
)

var (
	// Default options.
	defaultOptions = &Options{}
)

// Resource represents a test nats streaming server.
type Resource struct {
	// Nats streaming server docker container.
	*dockertest.Resource

	// Actual options.
	Options
}

// Options is options to run a nats streaming test server.
type Options struct {
	// Tag of the repository. Default: DefaultTag.
	Tag string

	// The cluster id of the server. Default: DefaultClusterId.
	ClusterId string

	// Use FILE store if true and use MEMORY store otherwise.
	// NOTE: Not support SQL store in this test server.
	FileStore bool

	// If specified and FileStore is true, data will be stored in this host directory.
	HostDataPath string

	// If specified, the port 4222/tcp will be mapped to it. Default: random port.
	HostPort uint16

	// If specified, the port 8222/tcp will be mapped to it. Default: random port.
	HostMonPort uint16

	// Expire time (in seconds) of the container. Default: DefaultExpire.
	Expire uint
}

// Run is equivalent to RunFromPool(nil, opts).
func Run(opts *Options) (*Resource, error) {
	return RunFromPool(nil, opts)
}

// RunFromPool runs a test nats streaming server. If pool is nil, tstsvc.DefaultPool() will be used.
// If opts is nil, the default options will be used.
func RunFromPool(pool *dockertest.Pool, opts *Options) (*Resource, error) {
	// Handle nil case.
	if pool == nil {
		pool = tstsvc.DefaultPool()
	}
	if opts == nil {
		opts = defaultOptions
	}

	// Collect options.
	res := &Resource{
		Options: *opts,
	}
	opts = &res.Options

	if opts.Tag == "" {
		opts.Tag = DefaultTag
	}
	if opts.ClusterId == "" {
		opts.ClusterId = DefaultClusterId
	}
	if opts.HostPort == 0 {
		opts.HostPort = tstsvc.FreePort()
	}
	if opts.HostMonPort == 0 {
		opts.HostMonPort = tstsvc.FreePort()
	}
	if opts.Expire == 0 {
		opts.Expire = DefaultExpire
	}

	// Run the container.
	runOpts := &dockertest.RunOptions{
		Repository: Repository,
		Tag:        opts.Tag,
		PortBindings: map[dc.Port][]dc.PortBinding{
			"4222/tcp": []dc.PortBinding{
				dc.PortBinding{
					HostIP:   "localhost",
					HostPort: fmt.Sprintf("%d", opts.HostPort),
				},
			},
			"8222/tcp": []dc.PortBinding{
				dc.PortBinding{
					HostIP:   "localhost",
					HostPort: fmt.Sprintf("%d", opts.HostMonPort),
				},
			},
		},
		Cmd: []string{"-cid", opts.ClusterId},
	}
	if opts.FileStore {
		runOpts.Cmd = append(runOpts.Cmd, "-st", "FILE", "--dir", "/data")
		if opts.HostDataPath != "" {
			runOpts.Mounts = append(runOpts.Mounts, fmt.Sprintf("%s:/data", opts.HostDataPath))
		}
	}

	var err error
	res.Resource, err = pool.RunWithOptions(runOpts)
	if err != nil {
		return nil, err
	}

	// Set expire of the container.
	res.Resource.Expire(opts.Expire)

	// Wait.
	if err := pool.Retry(func() error {
		sc, err := stan.Connect(
			res.Options.ClusterId,
			"6A05D2AB-7C75-4242-B345-A066439CE86E", // Hard code a random client id.
			stan.NatsURL(res.NatsURL()),
			stan.ConnectWait(100*time.Millisecond), // Shorter connect wait.
		)
		if err != nil {
			return err
		}
		sc.Close()
		return nil
	}); err != nil {
		res.Close()
		return nil, err
	}

	return res, nil
}

// NatsURL returns the nats url to connect to the nats streaming server.
func (res *Resource) NatsURL() string {
	return fmt.Sprintf("nats://localhost:%d", res.Options.HostPort)
}

// NatsClient returns a nats client of the embedded nats server of the test nats streaming server.
func (res *Resource) NatsClient(opts ...nats.Option) (*nats.Conn, error) {
	return nats.Connect(res.NatsURL(), opts...)
}

// StanClient returns a stan client of the test nats streaming server identified by clientId.
func (res *Resource) StanClient(clientId string, opts ...stan.Option) (stan.Conn, error) {
	opts = append(opts, stan.NatsURL(res.NatsURL()))
	return stan.Connect(res.Options.ClusterId, clientId, opts...)
}
