package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/chaos-mesh/go-sqlsmith/internal/dbutil"
	"github.com/chaos-mesh/go-sqlsmith/util"
	"github.com/go-sql-driver/mysql"
)

var (
	host                   = flag.String("host", "127.0.0.1", "Database host")
	port                   = flag.Int("port", 4000, "Database port")
	user                   = flag.String("user", "root", "Database user")
	passwd                 = flag.String("passwd", "", "Database password")
	dbname                 = flag.String("db", "test", "Database name")
	socket                 = flag.String("socket", "", "Database socket connect")
	configPath             = flag.String("config", "", "config path")
	clearDB                = flag.Bool("clear", false, "clear database and create a new one")
	conc                   = flag.Int("concurrency", 1, "concurrency, there may be error such as write conflict when it's larger than 1")
	passwordFile           = flag.String("password-file", "", "Read database password from a file (or use SQLSMITH_PASSWORD)")
	allowEmptyPassword     = flag.Bool("allow-empty-password", false, "Allow passwordless authentication to a disposable test instance")
	allowInsecureTransport = flag.Bool("allow-insecure-transport", false, "Allow plaintext TCP to an isolated remote test instance")
	tlsEnabled             = flag.Bool("tls", false, "Verify the database server using TLS")
	tlsCA                  = flag.String("tls-ca", "", "PEM CA file for database TLS")
	tlsServerName          = flag.String("tls-server-name", "", "Expected database TLS certificate name")
)

func log(l string, args ...interface{}) {
	fmt.Fprintln(os.Stderr, append([]interface{}{time.Now().UTC().Format(time.RFC3339), "[TEST]", l}, args...)...)
}

func logError(l string, args ...interface{}) {
	fmt.Fprintln(os.Stderr, append([]interface{}{time.Now().UTC().Format(time.RFC3339), "[ERROR]", l}, args...)...)
}

func getConfigs() (*mysql.Config, *mysql.Config, error) {
	return dbutil.Configs(dbutil.Options{
		Host: *host, Port: *port, User: *user, Password: *passwd,
		PasswordFile: *passwordFile, Database: *dbname, Socket: *socket,
		TLS: *tlsEnabled, TLSCA: *tlsCA, TLSServerName: *tlsServerName,
		AllowEmptyPassword: *allowEmptyPassword, AllowInsecureTransport: *allowInsecureTransport,
	})
}

func main() {
	flag.Parse()
	if *conc < 1 {
		logError("concurrency must be at least 1")
		os.Exit(1)
	}
	bootstrapConfig, databaseConfig, err := getConfigs()
	if err != nil {
		logError("database configuration", err)
		os.Exit(1)
	}
	config := NewSQLSmithConfig()
	if *configPath != "" {
		if err := config.Load(*configPath); err != nil {
			panic(err)
		}
	}

	log("connect", dbutil.Target(databaseConfig))
	connect, err := NewConnect(bootstrapConfig, *dbname, config)
	if err != nil {
		panic(err)
	}
	log("init environment")
	connect.Init()
	if *clearDB {
		log("init database")
		connect.MustExec(fmt.Sprintf(DB_DROP, util.QuoteIdentifier(*dbname)))
		connect.MustExec(fmt.Sprintf(DB_CREATE, util.QuoteIdentifier(*dbname)))
	}
	if err := connect.ReConnect(databaseConfig); err != nil {
		panic(err)
	}

	runCh := make(chan struct{}, 1)
	go func() {
		if err := connect.Run(*conc, runCh); err != nil {
			logError("running error", err)
		}
	}()

	sc := make(chan os.Signal, 1)
	signal.Notify(sc,
		os.Interrupt,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)
	fmt.Fprintf(os.Stderr, "Got signal %d to exit.\n", <-sc)

	runCh <- struct{}{}
}
