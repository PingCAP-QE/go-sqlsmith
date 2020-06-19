package main

import (
	"database/sql"
	"sync"

	"github.com/juju/errors"
)

type Executor struct {
	sync.Mutex
	db  *sql.DB
	txn *sql.Tx
}

type IExec interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
}

func (e *Executor) Begin() error {
	e.Lock()
	defer e.Unlock()
	if e.txn != nil {
		return nil
	}
	txn, err := e.db.Begin()
	if err != nil {
		return errors.Trace(err)
	}
	e.txn = txn
	return nil
}

func (e *Executor) Commit() error {
	e.Lock()
	defer func() {
		e.txn = nil
		e.Unlock()
	}()
	if e.txn == nil {
		return nil
	}
	return errors.Trace(e.txn.Commit())
}

func (e *Executor) Rollback() error {
	e.Lock()
	defer func() {
		e.txn = nil
		e.Unlock()
	}()
	if e.txn == nil {
		return nil
	}
	return errors.Trace(e.txn.Rollback())
}

func (e *Executor) GetExec() IExec {
	e.Lock()
	defer e.Unlock()
	if e.txn != nil {
		return e.txn
	}
	return e.db
}
