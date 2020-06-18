package main

import (
	"flag"
	"fmt"

	"github.com/chaos-mesh/go-sqlsmith"
	_ "github.com/go-sql-driver/mysql"
)

var (
	host   = flag.String("host", "127.0.0.1", "Database host")
	port   = flag.Int("port", 4000, "Database port")
	user   = flag.String("user", "root", "Database user")
	passwd = flag.String("passwd", "", "Database password")
	dbname = flag.String("db", "test", "Database name")
)

func init() {
	flag.Parse()
}

func log(l string, args ...interface{}) {
	fmt.Println(append([]interface{}{"[TEST]", l}, args...)...)
}

func logError(l string, args ...interface{}) {
	fmt.Println(append([]interface{}{"[ERROR]", l}, args...)...)
}

func main() {
	initDSN := fmt.Sprintf("%s:%s@tcp(%s:%d)/", *user, *passwd, *host, *port)
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", *user, *passwd, *host, *port, *dbname)

	log("connect to dsn", initDSN)
	connect, err := NewConnect(initDSN)
	if err != nil {
		panic(err)
	}
	log("init environment")
	connect.Init(dsn)
	log("init database")
	connect.MustExec(DB_DROP)
	connect.MustExec(DB_CREATE)

	log("create tables")
	connect.MustExec(PULLS_DROP)
	connect.MustExec(COMMENTS_DROP)
	connect.MustExec(PULLS_TABLE)
	connect.MustExec(COMMENTS_TABLE)

	ss := sqlsmith.New()
	ss.LoadSchema(schema, indexes)
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
			_, ok = tables[table]
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
