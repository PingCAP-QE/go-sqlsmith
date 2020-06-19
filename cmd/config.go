package main

import (
	"reflect"

	"github.com/BurntSushi/toml"
	"github.com/juju/errors"
)

const (
	TxnBegin           = "TxnBegin"
	TxnCommit          = "TxnCommit"
	TxnRollback        = "TxnRollback"
	DDLCreateTable     = "DDLCreateTable"
	DDLAlterTable      = "DDLAlterTable"
	DDLCreateIndex     = "DDLCreateIndex"
	DMLSelect          = "DMLSelect"
	DMLSelectForUpdate = "DMLSelectForUpdate"
	DMLDelete          = "DMLDelete"
	DMLUpdate          = "DMLUpdate"
	DMLInsert          = "DMLInsert"
)

var (
	Stmts = []string{
		TxnBegin,
		TxnCommit,
		TxnRollback,
		DDLCreateTable,
		DDLAlterTable,
		DDLCreateIndex,
		DMLSelect,
		DMLSelectForUpdate,
		DMLDelete,
		DMLUpdate,
		DMLInsert,
	}
)

// SQLSmith for sqlsmith generator configuration only
type SQLSmithConfig struct {
	TxnBegin           int `toml:"txn-begin"`
	TxnCommit          int `toml:"txn-commit"`
	TxnRollback        int `toml:"txn-rollback"`
	DDLCreateTable     int `toml:"ddl-create-table"`
	DDLAlterTable      int `toml:"ddl-alter-table"`
	DDLCreateIndex     int `toml:"ddl-create-index"`
	DMLSelect          int `toml:"dml-select"`
	DMLSelectForUpdate int `toml:"dml-select-for-update"`
	DMLDelete          int `toml:"dml-delete"`
	DMLUpdate          int `toml:"dml-update"`
	DMLInsert          int `toml:"dml-insert"`
}

func NewSQLSmithConfig() *SQLSmithConfig {
	return &SQLSmithConfig{
		TxnBegin:           20,
		TxnCommit:          20,
		TxnRollback:        10,
		DDLCreateTable:     1,
		DDLAlterTable:      10,
		DDLCreateIndex:     10,
		DMLSelect:          120,
		DMLSelectForUpdate: 30,
		DMLDelete:          10,
		DMLUpdate:          120,
		DMLInsert:          120,
	}
}

func (s *SQLSmithConfig) Load(path string) error {
	_, err := toml.DecodeFile(path, s)
	return errors.Trace(err)
}

func (s *SQLSmithConfig) GetField(field string) int {
	r := reflect.ValueOf(s)
	f := reflect.Indirect(r).FieldByName(field)
	return int(f.Int())
}
