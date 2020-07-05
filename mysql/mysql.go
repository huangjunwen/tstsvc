package tstmysql

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/go-sql-driver/mysql"
	"github.com/ory/dockertest"
	dc "github.com/ory/dockertest/docker"

	"github.com/huangjunwen/tstsvc"
)

var (
	// Docker repository.
	Repository = "mysql"

	// Default tag.
	DefaultTag = "8.0.19"

	// Default database name.
	DefaultDBName = "tst"

	// Default root password.
	DefaultRootPassword = "123456"

	// Default container expire time.
	DefaultExpire uint = 120
)

var (
	// Default options.
	defaultOptions = &Options{}
)

var (
	noopLogger mysql.Logger = nxNoopLogger{}
	// Copy from github.com/go-sql-driver/mysql/errors.go
	errLogger mysql.Logger = log.New(os.Stderr, "[mysql] ", log.Ldate|log.Ltime|log.Lshortfile)
)

// Resource represents a test MySQL server.
type Resource struct {
	// MySQL docker container.
	*dockertest.Resource

	// Actual options.
	Options
}

// Options is options to run a MySQL test server.
type Options struct {
	// Tag of the repository. Default: DefaultTag.
	Tag string

	// The database created when MySQL server starts. Default: DefaultDBName.
	DBName string

	// The root password. Default: DefaultRootPassword.
	RootPassword string

	// If specified, MySQL data will be mount to this host directory. Default: "".
	// NOTE: The directory must be either contain an existing MySQL database or completely empty.
	HostDataPath string

	// If specified, SQL files inside this host directory will be loaded when MySQL server initialize. Default: "".
	// NOTE: These files will not be loaded if HostDataPath is specified and contains an existing database.
	HostInitSQLPath string

	// If specified, the port 3306/tcp will be mapped to it. Default: random port.
	HostPort uint16

	// Expire time (in seconds) of the container. Default: DefaultExpire.
	Expire uint

	// BaseRunOptions is the base options, will be overrided by above.
	BaseRunOptions dockertest.RunOptions
}

type nxNoopLogger struct{}

func (l nxNoopLogger) Print(v ...interface{}) {}

// Run is equivalent to RunFromPool(nil, opts).
func Run(opts *Options) (*Resource, error) {
	return RunFromPool(nil, opts)
}

// RunFromPool runs a test MySQL server. If pool is nil, tstsvc.DefaultPool() will be used.
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
	if opts.DBName == "" {
		opts.DBName = DefaultDBName
	}
	if opts.RootPassword == "" {
		opts.RootPassword = DefaultRootPassword
	}
	if opts.HostPort == 0 {
		opts.HostPort = tstsvc.FreePort()
	}
	if opts.Expire == 0 {
		opts.Expire = DefaultExpire
	}

	// Copy and collect RunOptions.
	runOpts := opts.BaseRunOptions
	runOpts.Env = append([]string(nil), runOpts.Env...)
	runOpts.Mounts = append([]string(nil), runOpts.Mounts...)

	if runOpts.Repository == "" {
		runOpts.Repository = Repository
	}
	runOpts.Tag = opts.Tag
	runOpts.Env = append(runOpts.Env,
		fmt.Sprintf("MYSQL_DATABASE=%s", opts.DBName),
		fmt.Sprintf("MYSQL_ROOT_PASSWORD=%s", opts.RootPassword),
	)
	if opts.HostInitSQLPath != "" {
		runOpts.Mounts = append(runOpts.Mounts, fmt.Sprintf("%s:/docker-entrypoint-initdb.d", opts.HostInitSQLPath))
	}
	if opts.HostDataPath != "" {
		runOpts.Mounts = append(runOpts.Mounts, fmt.Sprintf("%s:/var/lib/mysql", opts.HostDataPath))
	}
	runOpts.PortBindings = map[dc.Port][]dc.PortBinding{
		"3306/tcp": []dc.PortBinding{
			dc.PortBinding{
				HostIP:   "localhost",
				HostPort: fmt.Sprintf("%d", opts.HostPort),
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

	// Suppress error output when waiting server up.
	mysql.SetLogger(noopLogger)
	defer mysql.SetLogger(errLogger)

	// Wait.
	if err := pool.Retry(func() error {
		db, err := res.Client()
		if err != nil {
			return err
		}
		defer db.Close()
		return db.Ping()
	}); err != nil {
		res.Close()
		return nil, err
	}

	return res, nil
}

// DSN returns the data source name of the test MySQL server.
func (res *Resource) DSN() string {
	return fmt.Sprintf(
		"root:%s@tcp(localhost:%d)/%s?parseTime=true",
		res.Options.RootPassword,
		res.Options.HostPort,
		res.Options.DBName,
	)
}

// Client returns a client to the test MySQL server.
func (res *Resource) Client() (*sql.DB, error) {
	return sql.Open("mysql", res.DSN())
}
