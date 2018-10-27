package tstmysql

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/huangjunwen/tstsvc/tstsvc"

	"github.com/go-sql-driver/mysql"
	"github.com/ory/dockertest"
	dc "github.com/ory/dockertest/docker"
)

var (
	// Docker repository.
	Repository = "mysql"

	// Default tag.
	DefaultTag = "5.7.21"

	// Default database name.
	DefaultDatabaseName = "tst"

	// Default root password.
	DefaultRootPassword = "123456"

	// Default container expire time.
	DefaultExpire uint = 120
)

var (
	// Default options.
	DefaultOptions = &Options{}
)

var (
	noopLogger mysql.Logger = nxNoopLogger{}
	// Copy from github.com/go-sql-driver/mysql/errors.go
	errLogger mysql.Logger = log.New(os.Stderr, "[mysql] ", log.Ldate|log.Ltime|log.Lshortfile)
)

// Options is options to run a MySQL test server.
type Options struct {
	// Tag of the repository. Default: DefaultTag.
	Tag string

	// The database created when MySQL server starts. Default: DefaultDatabaseName.
	DatabaseName string

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
}

type nxNoopLogger struct{}

func (l nxNoopLogger) Print(v ...interface{}) {}

// Run is equivalent to RunFromPool(nil).
func (o *Options) Run() (res *dockertest.Resource, dsn string, err error) {
	return o.RunFromPool(nil)
}

// RunFromPool runs a MySQL test server. If pool is nil, tstsvc.DefaultPool() will be used.
func (o *Options) RunFromPool(pool *dockertest.Pool) (res *dockertest.Resource, dsn string, err error) {
	// Get pool.
	if pool == nil {
		pool = tstsvc.DefaultPool()
	}

	// Collect run options.
	opts := &dockertest.RunOptions{
		Repository:   Repository,
		PortBindings: map[dc.Port][]dc.PortBinding{},
	}

	tag := o.Tag
	if tag == "" {
		tag = DefaultTag
	}
	opts.Tag = tag

	databaseName := o.DatabaseName
	if databaseName == "" {
		databaseName = DefaultDatabaseName
	}
	opts.Env = append(opts.Env, fmt.Sprintf("MYSQL_DATABASE=%s", databaseName))

	rootPassword := o.RootPassword
	if rootPassword == "" {
		rootPassword = DefaultRootPassword
	}
	opts.Env = append(opts.Env, fmt.Sprintf("MYSQL_ROOT_PASSWORD=%s", rootPassword))

	if o.HostInitSQLPath != "" {
		opts.Mounts = append(opts.Mounts, fmt.Sprintf("%s:/docker-entrypoint-initdb.d", o.HostInitSQLPath))
	}

	if o.HostDataPath != "" {
		opts.Mounts = append(opts.Mounts, fmt.Sprintf("%s:/var/lib/mysql", o.HostDataPath))
	}

	if o.HostPort != 0 {
		opts.PortBindings["3306/tcp"] = []dc.PortBinding{
			dc.PortBinding{
				HostIP:   "localhost",
				HostPort: fmt.Sprintf("%d", o.HostPort),
			},
		}
	}

	// Now starts the container.
	res, err = pool.RunWithOptions(opts)
	if err != nil {
		return nil, "", err
	}

	// Set expire of the container.
	expire := o.Expire
	if expire == 0 {
		expire = DefaultExpire
	}
	res.Expire(expire)

	// Suppress error output when waiting server up.
	mysql.SetLogger(noopLogger)
	defer mysql.SetLogger(errLogger)

	// Format data source name.
	dsn = fmt.Sprintf(
		"root:%s@tcp(localhost:%s)/%s?parseTime=true",
		rootPassword,
		res.GetPort("3306/tcp"),
		databaseName,
	)

	// Wait.
	if err := pool.Retry(func() error {
		db, err := sql.Open("mysql", dsn)
		if err != nil {
			return err
		}
		defer db.Close()
		return db.Ping()
	}); err != nil {
		res.Close()
		return nil, "", err
	}

	return res, dsn, nil

}

// Run is equivalent to DefaultOptions.Run().
func Run() (res *dockertest.Resource, dsn string, err error) {
	return DefaultOptions.Run()
}

// RunFromPool is equivalent to DefaultOptions.RunFromPool(pool).
func RunFromPool(pool *dockertest.Pool) (res *dockertest.Resource, dsn string, err error) {
	return DefaultOptions.RunFromPool(pool)
}
