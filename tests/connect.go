package main

import (
	"database/sql"
	"time"
)

type Connect struct {
	dsn string
	db  *sql.DB
}

func NewConnect(dsn string) (*Connect, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	return &Connect{
		dsn: dsn,
		db:  db,
	}, nil
}

func (c *Connect) MustExec(query string, args ...interface{}) {
	_, err := c.db.Exec(query, args...)
	if err != nil {
		logError(query, args)
		panic(err)
	}
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
