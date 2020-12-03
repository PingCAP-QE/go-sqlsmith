package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/go-sql-driver/mysql"
)

var (
	host       = flag.String("host", "127.0.0.1", "Database host")
	port       = flag.Int("port", 4000, "Database port")
	user       = flag.String("user", "root", "Database user")
	passwd     = flag.String("passwd", "", "Database password")
	dbname     = flag.String("db", "test", "Database name")
	socket     = flag.String("socket", "", "Database socket connect")
	configPath = flag.String("config", "", "config path")
	clearDB    = flag.Bool("clear", false, "clear database and create a new one")
	conc       = flag.Int("concurrency", 1, "concurrency, there may be error such as write conflict when it's larger than 1")
)

func init() {
	flag.Parse()
}

func log(l string, args ...interface{}) {
	fmt.Fprintln(os.Stderr, append([]interface{}{"[TEST]", l}, args...)...)
}

func logError(l string, args ...interface{}) {
	fmt.Fprintln(os.Stderr, append([]interface{}{"[ERROR]", l}, args...)...)
}

func getDSN() (string, string) {
	var (
		dsn     string
		initDSN string
	)
	if *socket == "" {
		initDSN = fmt.Sprintf("%s:%s@tcp(%s:%d)/", *user, *passwd, *host, *port)
	} else {
		initDSN = fmt.Sprintf("%s:%s@unix(%s)/", *user, *passwd, *socket)
	}
	dsn = fmt.Sprintf("%s%s", initDSN, *dbname)
	return initDSN, dsn
}

func main() {
	initDSN, dsn := getDSN()
	config := NewSQLSmithConfig()
	if *configPath != "" {
		if err := config.Load(*configPath); err != nil {
			panic(err)
		}
	}

	log("connect to dsn", initDSN)
	connect, err := NewConnect(initDSN, *dbname, config)
	if err != nil {
		panic(err)
	}
	log("init environment")
	connect.Init(initDSN)
	if *clearDB {
		log("init database")
		connect.MustExec(fmt.Sprintf(DB_DROP, *dbname))
		connect.MustExec(fmt.Sprintf(DB_CREATE, *dbname))
	}
	if err := connect.ReConnect(dsn); err != nil {
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
