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

func (c *Connect) ReConnect() error {
	db, err := sql.Open("mysql", c.dsn)
	if err != nil {
		return err
	}
	c.db = db
	return nil
}

func (c *Connect) Init() {
	c.MustExec(SQL_MODE)
	time.Sleep(5 * time.Second)
	if err := c.ReConnect(); err != nil {
		panic(err)
	}
}
