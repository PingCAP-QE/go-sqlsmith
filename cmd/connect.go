package main

import (
	"database/sql"
	"fmt"
	"math/rand"
	"time"

	"github.com/chaos-mesh/go-sqlsmith"
	"github.com/juju/errors"
)

const (
	indexColumnName = "Key_name"
)

type Connect struct {
	dsn      string
	db       *sql.DB
	dbs      []*Executor
	config   *SQLSmithConfig
	dbname   string
	ss       *sqlsmith.SQLSmith
	sum      int
	points   []int
	num2stmt map[int]string
	stmt2num map[string]int
}

func NewConnect(dsn string, dbname string, config *SQLSmithConfig) (*Connect, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	ss := sqlsmith.New()
	ss.SetDB(dbname)
	c := &Connect{
		dsn:      dsn,
		db:       db,
		config:   config,
		dbname:   dbname,
		ss:       ss,
		points:   make([]int, len(Stmts)+1),
		num2stmt: make(map[int]string),
		stmt2num: make(map[string]int),
	}
	if err := c.InitSmithRates(); err != nil {
		return nil, errors.Trace(err)
	}
	return c, nil
}

func (c *Connect) ReConnect(dsn string) error {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return err
	}
	c.dsn = dsn
	c.db = db
	return nil
}

func (c *Connect) Init(dsn string) {
	c.MustExec(SQL_MODE)
	time.Sleep(5 * time.Second)
	c.dsn = dsn
	if err := c.ReConnect(dsn); err != nil {
		panic(err)
	}
}

func (c *Connect) MustExec(query string, args ...interface{}) {
	_, err := c.db.Exec(query, args...)
	if err != nil {
		logError(query, args)
		panic(err)
	}
}

func (c *Connect) ReloadSchema() error {
	schema, err := c.FetchSchema(c.dbname)
	if err != nil {
		return errors.Trace(err)
	}
	indexes := make(map[string][]string)
	for _, col := range schema {
		table := col[1]
		if _, ok := indexes[table]; ok {
			continue
		}
		tableIndexes, err := c.FetchIndexes(c.dbname, table)
		if err != nil {
			return errors.Trace(err)
		}
		indexes[table] = tableIndexes
	}
	c.ss.LoadSchema(schema, indexes)
	return nil
}

// Exec runs query and return result
// TODO: read result rows
func (c *Connect) Exec(query string, args ...interface{}) (interface{}, error) {
	_, err := c.db.Exec(query, args...)
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func (c *Connect) FetchSchema(db string) ([][5]string, error) {
	var (
		schema     [][5]string
		tablesInDB [][3]string
	)
	tables, err := c.db.Query(schemaSQL)
	if err != nil {
		return nil, errors.Trace(err)
	}

	// fetch tables need to be described
	for tables.Next() {
		var schemaName, tableName, tableType string
		if err = tables.Scan(&schemaName, &tableName, &tableType); err != nil {
			return [][5]string{}, errors.Trace(err)
		}
		if schemaName == db {
			tablesInDB = append(tablesInDB, [3]string{schemaName, tableName, tableType})
		}
	}

	// desc tables
	for _, table := range tablesInDB {
		var (
			schemaName = table[0]
			tableName  = table[1]
			tableType  = table[2]
		)
		columns, err := c.FetchColumns(schemaName, tableName)
		if err != nil {
			return [][5]string{}, errors.Trace(err)
		}
		for _, column := range columns {
			schema = append(schema, [5]string{schemaName, tableName, tableType, column[0], column[1]})
		}
	}
	return schema, nil
}

// FetchColumns get columns for given table
func (c *Connect) FetchColumns(db, table string) ([][2]string, error) {
	var columns [][2]string
	res, err := c.db.Query(fmt.Sprintf(tableSQL, db, table))
	if err != nil {
		return [][2]string{}, errors.Trace(err)
	}
	for res.Next() {
		var columnName, columnType string
		var col1, col2, col3, col4 interface{}
		if err = res.Scan(&columnName, &columnType, &col1, &col2, &col3, &col4); err != nil {
			return [][2]string{}, errors.Trace(err)
		}
		columns = append(columns, [2]string{columnName, columnType})
	}
	return columns, nil
}

// FetchIndexes get indexes for given table
func (c *Connect) FetchIndexes(db, table string) ([]string, error) {
	var indexes []string
	res, err := c.db.Query(fmt.Sprintf(indexSQL, db, table))
	if err != nil {
		return []string{}, errors.Trace(err)
	}

	columnTypes, err := res.ColumnTypes()
	if err != nil {
		return indexes, errors.Trace(err)
	}
	for res.Next() {
		var (
			keyname       string
			rowResultSets []interface{}
		)

		for range columnTypes {
			rowResultSets = append(rowResultSets, new(interface{}))
		}
		if err = res.Scan(rowResultSets...); err != nil {
			return []string{}, errors.Trace(err)
		}

		for index, resultItem := range rowResultSets {
			if columnTypes[index].Name() != indexColumnName {
				continue
			}
			r := *resultItem.(*interface{})
			if r != nil {
				bytes := r.([]byte)
				keyname = string(bytes)
			}
		}

		if keyname != "" && keyname != "PRIMARY" {
			indexes = append(indexes, keyname)
		}
	}
	return indexes, nil
}

func (c *Connect) forkDB(concurrency int) error {
	c.dbs = make([]*Executor, concurrency)
	for i := 0; i < concurrency; i++ {
		db, err := sql.Open("mysql", c.dsn)
		if err != nil {
			return err
		}
		c.dbs[i] = &Executor{db: db}
	}
	return nil
}

func (c *Connect) InitTables() error {
	for i := 0; i < 10; i++ {
		stmt, _, err := c.ss.CreateTableStmt()
		if err != nil {
			return errors.Trace(err)
		}
		if _, err := c.Exec(stmt); err != nil {
			return errors.Trace(err)
		}
	}
	return errors.Trace(c.ReloadSchema())
}

// Run starts running SQLSmith
func (c *Connect) Run(concurrency int, ch <-chan struct{}) error {
	if concurrency < 1 {
		return errors.New("concurrency should be at least 1")
	}
	if err := c.forkDB(concurrency); err != nil {
		return errors.Trace(err)
	}
	if err := c.InitTables(); err != nil {
		return errors.Trace(err)
	}
	var (
		stopCh = make(chan struct{}, concurrency)
		doneCh = make(chan struct{}, concurrency)
		errCh  = make(chan error, 1)
	)

	stop := func() {
		for i := 0; i < concurrency; i++ {
			stopCh <- struct{}{}
		}
		for i := 0; i < concurrency; i++ {
			<-doneCh
		}
	}

	for i := 0; i < concurrency; i++ {
		go c.run(i, stopCh, doneCh, errCh)
	}
	go func() {
		for {
			logError("stmt exec error", <-errCh)
		}
	}()

	<-ch
	stop()
	return nil
}

func (c *Connect) run(i int, stopCh <-chan struct{}, doneCh chan struct{}, errCh chan error) {
	for {
		select {
		case <-stopCh:
			doneCh <- struct{}{}
			return
		default:
			if err := c.RunOnce(i); err != nil {
				errCh <- err
			}
		}
	}
}

// RunOnce ...
func (c *Connect) RunOnce(i int) error {
	var (
		stmt string
		err  error
	)
	var (
		exec = c.dbs[i]
		tp   = c.RandStmt()
	)
	switch tp {
	case TxnBegin:
		return errors.Trace(exec.Begin())
	case TxnCommit:
		return errors.Trace(exec.Commit())
	case TxnRollback:
		return errors.Trace(exec.Rollback())
	case DDLCreateTable:
		stmt, _, err = c.ss.CreateTableStmt()
	case DDLAlterTable:
		stmt, err = c.ss.AlterTableStmt(&sqlsmith.DDLOptions{})
	case DDLCreateIndex:
		stmt, err = c.ss.CreateIndexStmt(&sqlsmith.DDLOptions{})
	case DMLSelect:
		stmt, _, err = c.ss.SelectStmt(1 + rand.Intn(4))
	case DMLSelectForUpdate:
		stmt, _, err = c.ss.SelectForUpdateStmt(1 + rand.Intn(4))
	case DMLDelete:
		stmt, _, err = c.ss.DeleteStmt()
	case DMLUpdate:
		stmt, _, err = c.ss.UpdateStmt()
	case DMLInsert:
		stmt, _, err = c.ss.InsertStmt(false)
	}

	if err != nil {
		return errors.Trace(err)
	}
	_, err = exec.GetExec().Exec(stmt)
	if err != nil {
		return errors.Trace(err)
	}
	switch tp {
	case DDLCreateTable, DDLAlterTable, DDLCreateIndex:
		return errors.Trace(c.ReloadSchema())
	default:
		return nil
	}
}

func (c *Connect) InitSmithRates() error {
	sum := 0
	for _, stmt := range Stmts {
		p := c.config.GetField(stmt)
		if p == 0 {
			continue
		}
		c.num2stmt[sum] = stmt
		c.stmt2num[stmt] = sum
		c.points = append(c.points, sum)
		sum += p
	}
	c.points = append(c.points, sum)
	if sum == 0 {
		return errors.New("possibilities' sum is 0")
	}
	c.sum = sum
	return nil
}

func (c *Connect) RandStmt() string {
	rd := rand.Intn(c.sum)
	for i := range c.points {
		if rd >= c.points[i] && rd < c.points[i+1] {
			return c.num2stmt[c.points[i]]
		}
	}
	panic("unreachable")
}
