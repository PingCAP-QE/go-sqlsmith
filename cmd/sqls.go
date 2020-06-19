package main

const (
	SQL_MODE = `SET @@GLOBAL.SQL_MODE="NO_ENGINE_SUBSTITUTION"`

	DB_DROP   = `DROP DATABASE IF EXISTS %s`
	DB_CREATE = `CREATE DATABASE %s`

	schemaSQL = "SELECT TABLE_SCHEMA, TABLE_NAME, TABLE_TYPE FROM information_schema.tables"
	tableSQL  = "DESC %s.%s"
	indexSQL  = "SHOW INDEX FROM %s.%s"
)
