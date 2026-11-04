package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/chaos-mesh/go-sqlsmith"
	"github.com/chaos-mesh/go-sqlsmith/internal/dbutil"
	"github.com/chaos-mesh/go-sqlsmith/util"
	"github.com/go-sql-driver/mysql"
)

var (
	clearDB                = flag.Bool("clear", false, "Explicitly allow integration tests to replace the selected database")
	host                   = flag.String("host", "127.0.0.1", "Database host")
	port                   = flag.Int("port", 4000, "Database port")
	user                   = flag.String("user", "root", "Database user")
	passwd                 = flag.String("passwd", "", "Database password")
	dbname                 = flag.String("db", "test", "Database name")
	socket                 = flag.String("socket", "", "Database socket connect")
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
	if !*clearDB {
		logError("integration tests replace the selected database; pass -clear only for a disposable database")
		os.Exit(1)
	}
	bootstrapConfig, databaseConfig, err := getConfigs()
	if err != nil {
		logError("database configuration", err)
		os.Exit(1)
	}

	log("connect", dbutil.Target(databaseConfig))
	connect, err := NewConnect(bootstrapConfig)
	if err != nil {
		panic(err)
	}
	log("init environment")
	connect.Init()
	log("init database")
	connect.MustExec(fmt.Sprintf(DB_DROP, util.QuoteIdentifier(*dbname)))
	connect.MustExec(fmt.Sprintf(DB_CREATE, util.QuoteIdentifier(*dbname)))
	if err := connect.ReConnect(databaseConfig); err != nil {
		panic(err)
	}

	log("create tables")
	connect.MustExec(PULLS_DROP)
	connect.MustExec(COMMENTS_DROP)
	connect.MustExec(PULLS_TABLE)
	connect.MustExec(COMMENTS_TABLE)

	ss := sqlsmith.New()
	testSchema := append([][5]string(nil), schema...)
	for i := range testSchema {
		testSchema[i][0] = *dbname
	}
	ss.LoadSchema(testSchema, indexes)
	ss.SetDB(*dbname)

	// insert data
	log("exec insert test")
	for i := 0; i < 100; i++ {
		// simple insert
		insertSQL, _, err := ss.InsertStmt(false)
		if err != nil {
			logError(insertSQL)
			panic(err)
		}
		connect.MustExec(insertSQL)
		// insert with function
		insertSQL, _, err = ss.InsertStmt(false)
		if err != nil {
			logError(insertSQL)
			panic(err)
		}
		connect.MustExec(insertSQL)
	}
	// update data
	log("exec update test")
	for i := 0; i < 100; i++ {
		updateSQL, _, err := ss.UpdateStmt()
		if err != nil {
			logError(updateSQL)
			panic(err)
		}
		connect.MustExec(updateSQL)
	}
	// delete & insert data
	log("exec delete test")
	for i := 0; i < 100; i++ {
		deleteSQL, _, err := ss.DeleteStmt()
		if err != nil {
			logError(deleteSQL)
			panic(err)
		}
		connect.MustExec(deleteSQL)
		insertSQL, _, err := ss.InsertStmt(false)
		if err != nil {
			logError(insertSQL)
			panic(err)
		}
		connect.MustExec(insertSQL)
	}
	// create table
	log("exec create table")
	tables := map[string]struct{}{
		"": {},
	}
	for i := 0; i < 100; i++ {
		var (
			createSQL string
			table     string
			err       error
			ok        = false
		)
		for !ok {
			createSQL, table, err = ss.CreateTableStmt()
			if err != nil {
				logError(createSQL)
				panic(err)
			}
			_, exists := tables[table]
			ok = !exists
			tables[table] = struct{}{}
		}
		connect.MustExec(createSQL)
	}
	// alter table
	// this will make schema changed, so we perform it only once
	log("exec alter table")
	alterSQL, err := ss.AlterTableStmt(&sqlsmith.DDLOptions{OnlineDDL: true})
	if err != nil {
		logError(alterSQL)
		panic(err)
	}
	connect.MustExec(alterSQL)
}
