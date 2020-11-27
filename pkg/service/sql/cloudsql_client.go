package sql

import (
    "fmt"
    "github.com/go-pg/pg/v10"
    "strconv"
    "sync"
)

var instance CloudSQLClient
var once sync.Once

//ConnectOptions contains parameters for connecting to the database
type ConnectOptions struct {
    Host         string
    Socket       string
    Port         string
    User         string
    Password     string
    Database     string
    DebugQueries string
}

func (co *ConnectOptions) toPgOptions() *pg.Options {
    network := "unix"
    addr := co.Socket
    if addr == "" {
        network = "tcp"
        addr = fmt.Sprintf("%s:%s", co.Host, co.Port)
    }
    return &pg.Options{
        Addr:     addr,
        Network:  network,
        User:     co.User,
        Password: co.Password,
        Database: co.Database,
    }
}

// CloudSQLClient defines db interface
type CloudSQLClient interface {
    IsInitialized() bool
    DB() *pg.DB
    Close() error
}

// defaultCloudSQLClient defines a db to a db
type defaultCloudSQLClient struct {
    db    *pg.DB
    debug bool
}

// NewCloudSQLClient crete new instance of CloudSQLClient
func NewCloudSQLClient(options ConnectOptions) CloudSQLClient {
    once.Do(func() {
        db := pg.Connect(options.toPgOptions())
        debug, _ := strconv.ParseBool(options.DebugQueries)
        instance = &defaultCloudSQLClient{db: db, debug: debug}
    })
    return instance
}

// IsInitialized check if db is present
func (c *defaultCloudSQLClient) IsInitialized() bool {
    return c.db != nil
}

// DB will create a new db as long current ones don't exceed configured value
func (c *defaultCloudSQLClient) DB() *pg.DB {
    return c.db
}

// Close closes db connection
func (c *defaultCloudSQLClient) Close() error {
    return c.db.Close()
}
