package main

const (
	DB_DROP   = `DROP DATABASE IF EXISTS %s`
	DB_CREATE = `CREATE DATABASE %s`

	schemaSQL = "SELECT TABLE_SCHEMA, TABLE_NAME, TABLE_TYPE FROM information_schema.tables WHERE TABLE_SCHEMA = ?"
	tableSQL  = "DESC %s.%s"
	indexSQL  = "SHOW INDEX FROM %s.%s"
)
