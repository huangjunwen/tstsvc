package tstmysql

import (
	"database/sql"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/ory/dockertest"
	"github.com/stretchr/testify/assert"
)

func TestRun(t *testing.T) {
	assert := assert.New(t)
	var err error

	// Create temp directory.
	tmpDir, err := ioutil.TempDir("/tmp", "tstmysql")
	if err != nil {
		log.Panic(err)
	}
	defer os.RemoveAll(tmpDir)
	log.Printf("Temp directory created: %s\n", tmpDir)

	// Create init directory.
	initDir := filepath.Join(tmpDir, "initdb")
	if err := os.Mkdir(initDir, 0777); err != nil {
		log.Panic(err)
	}
	log.Printf("Init db directory created: %s\n", initDir)

	dataDir := filepath.Join(tmpDir, "data")
	if err := os.Mkdir(dataDir, 0777); err != nil {
		log.Panic(err)
	}
	log.Printf("Data directory created: %s\n", dataDir)

	// Create init sql.
	initSQL, err := os.OpenFile(filepath.Join(initDir, "tstmysql.sql"), os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		log.Panic(err)
	}
	defer initSQL.Close()

	// Write init sql.
	_, err = initSQL.WriteString(`
			CREATE TABLE xxx (
				id INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
				name VARCHAR(64) NOT NULL
			);

			INSERT INTO xxx (name) VALUES ("ada");
		`)
	if err != nil {
		log.Panic(err)
	}
	log.Printf("Init SQL created.\n")

	// Use the above host mounts.
	opts := &Options{
		HostInitSQLPath: initDir,
		HostDataPath:    dataDir,
	}

	// Run the first server.
	var res1 *dockertest.Resource
	var dsn1 string
	{
		res1, dsn1, err = opts.Run()
		assert.NoError(err)
		defer res1.Close()
	}
	log.Printf("The first MySQL server is up, DSN: %+q.\n", dsn1)

	// Connect to the first server.
	var db1 *sql.DB
	{
		db1, err = sql.Open("mysql", dsn1)
		assert.NoError(err)
		defer db1.Close()
	}
	log.Printf("Connected to the first MySQL server.\n")

	// The init sql should have been loaded and have one row.
	{
		var n sql.NullInt64
		err := db1.QueryRow("SELECT COUNT(*) FROM xxx").Scan(&n)
		assert.NoError(err)
		assert.True(n.Valid)
		assert.Equal(int64(1), n.Int64)
	}

	// Add one more row.
	{
		_, err := db1.Exec("INSERT INTO xxx (name) VALUES ('bob')")
		assert.NoError(err)
	}

	// Stop the first server.
	res1.Close()
	db1.Close()
	log.Printf("The first MySQL server is down.\n")

	// Run the second server.
	var res2 *dockertest.Resource
	var dsn2 string
	{
		res2, dsn2, err = opts.Run()
		assert.NoError(err)
		defer res2.Close()
	}
	log.Printf("The second MySQL server is up, DSN: %+q.\n", dsn2)

	// Connect to the second server.
	var db2 *sql.DB
	{
		db2, err = sql.Open("mysql", dsn2)
		assert.NoError(err)
		defer db2.Close()
	}
	log.Printf("Connected to the second MySQL server.\n")

	// The init sql should not load. And should have 2 rows.
	{
		var n sql.NullInt64
		err := db2.QueryRow("SELECT COUNT(*) FROM xxx").Scan(&n)
		assert.NoError(err)
		assert.True(n.Valid)
		assert.Equal(int64(2), n.Int64)
	}
}
