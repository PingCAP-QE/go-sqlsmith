package main

import (
	"database/sql"
	"fmt"

	"github.com/chaos-mesh/go-sqlsmith/internal/dbutil"
	"github.com/go-sql-driver/mysql"
)

type Connect struct {
	dbConfig *mysql.Config
	db       *sql.DB
}

func NewConnect(dbConfig *mysql.Config) (*Connect, error) {
	db, err := dbutil.Open(dbConfig)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(2)
	return &Connect{
		dbConfig: dbConfig,
		db:       db,
	}, nil
}

func (c *Connect) MustExec(query string, args ...interface{}) {
	_, err := c.db.Exec(query, args...)
	if err != nil {
		logError("statement failed", dbutil.Target(c.dbConfig), fmt.Sprintf("sql=%q", query))
		panic(err)
	}
	log("statement succeeded", dbutil.Target(c.dbConfig), fmt.Sprintf("sql=%q", query))
}

func (c *Connect) ReConnect(dbConfig *mysql.Config) error {
	db, err := dbutil.Open(dbConfig)
	if err != nil {
		return err
	}
	db.SetMaxOpenConns(2)
	if c.db != nil {
		c.db.Close()
	}
	c.dbConfig = dbConfig
	c.db = db
	return nil
}

func (c *Connect) Init() {
	if err := c.db.Ping(); err != nil {
		panic(err)
	}
}
