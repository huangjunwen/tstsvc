package tstnats

import (
	"fmt"

	nats "github.com/nats-io/nats.go"
	"github.com/ory/dockertest"
	dc "github.com/ory/dockertest/docker"

	"github.com/huangjunwen/tstsvc"
)

var (
	// Docker repository.
	Repository = "nats"

	// Default tag.
	DefaultTag = "2.0.0-linux"

	// Default container expire time.
	DefaultExpire uint = 120
)

var (
	// Default options.
	defaultOptions = &Options{}
)

// Resource represents a test nats server.
type Resource struct {
	// Nats streaming server docker container.
	*dockertest.Resource

	// Actual options.
	Options
}

// Options is options to run a test nats server.
type Options struct {
	// Tag of the repository. Default: DefaultTag.
	Tag string

	// If specified, the port 4222/tcp will be mapped to it. Default: random port.
	HostPort uint16

	// If specified, the port 8222/tcp will be mapped to it. Default: random port.
	HostMonPort uint16

	// If specified, the port 6222/tcp will be mapped to it. Default: random port.
	HostClusterPort uint16

	// Expire time (in seconds) of the container. Default: DefaultExpire.
	Expire uint

	// BaseRunOptions is the base options, will be overrided by above.
	BaseRunOptions dockertest.RunOptions
}

// Run is equivalent to RunFromPool(nil, opts).
func Run(opts *Options) (*Resource, error) {
	return RunFromPool(nil, opts)
}

// RunFromPool runs a test nats server. If pool is nil, tstsvc.DefaultPool() will be used.
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
	if opts.HostPort == 0 {
		opts.HostPort = tstsvc.FreePort()
	}
	if opts.HostMonPort == 0 {
		opts.HostMonPort = tstsvc.FreePort()
	}
	if opts.HostClusterPort == 0 {
		opts.HostClusterPort = tstsvc.FreePort()
	}
	if opts.Expire == 0 {
		opts.Expire = DefaultExpire
	}

	// Copy and collect RunOptions.
	runOpts := opts.BaseRunOptions

	if runOpts.Repository == "" {
		runOpts.Repository = Repository
	}
	runOpts.Tag = opts.Tag
	runOpts.PortBindings = map[dc.Port][]dc.PortBinding{
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
		"6222/tcp": []dc.PortBinding{
			dc.PortBinding{
				HostIP:   "localhost",
				HostPort: fmt.Sprintf("%d", opts.HostClusterPort),
			},
		},
	}

	var err error
	res.Resource, err = pool.RunWithOptions(&runOpts)
	if err != nil {
		return nil, err
	}

	// Set expire of the container.
	res.Resource.Expire(opts.Expire)

	// Wait.
	if err := pool.Retry(func() error {
		nc, err := nats.Connect(res.NatsURL())
		if err != nil {
			return err
		}
		nc.Close()
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
