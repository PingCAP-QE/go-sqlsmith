package main

const (
	SQL_MODE = `SET @@GLOBAL.SQL_MODE="NO_ENGINE_SUBSTITUTION"`

	DB_DROP   = `DROP DATABASE IF EXISTS test`
	DB_CREATE = `CREATE DATABASE test`

	PULLS_DROP  = `DROP TABLE IF EXISTS pulls`
	PULLS_TABLE = `CREATE TABLE pulls (
		id int(11) NOT NULL AUTO_INCREMENT,
		owner varchar(255) DEFAULT NULL,
		repo varchar(255) DEFAULT NULL,
		pull_number int(11) DEFAULT NULL,
		title text,
		body text,
		created_at timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (id),
		KEY index_pull_number (pull_number)
	)`
	COMMENTS_DROP  = `DROP TABLE IF EXISTS comments`
	COMMENTS_TABLE = `CREATE TABLE comments (
		id int(11) NOT NULL AUTO_INCREMENT,
		owner varchar(255) DEFAULT NULL,
		repo varchar(255) DEFAULT NULL,
		comment_id int(11) DEFAULT NULL,
		pull_number int(11) DEFAULT NULL,
		body text,
		user varchar(255) DEFAULT NULL,
		created_at timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (id),
		KEY index_comments_pull_number (pull_number)
	)`
)

var schema = [][5]string{
	{"test", "pulls", "BASE TABLE", "id", "int"},
	{"test", "pulls", "BASE TABLE", "owner", "varchar"},
	{"test", "pulls", "BASE TABLE", "repo", "varchar"},
	{"test", "pulls", "BASE TABLE", "pull_number", "int"},
	{"test", "pulls", "BASE TABLE", "title", "text"},
	{"test", "pulls", "BASE TABLE", "body", "text"},
	{"test", "pulls", "BASE TABLE", "created_at", "timestamp"},
	{"test", "comments", "BASE TABLE", "id", "int"},
	{"test", "comments", "BASE TABLE", "owner", "varchar"},
	{"test", "comments", "BASE TABLE", "repo", "varchar"},
	{"test", "comments", "BASE TABLE", "comment_id", "int"},
	{"test", "comments", "BASE TABLE", "pull_number", "int"},
	{"test", "comments", "BASE TABLE", "body", "text"},
	{"test", "comments", "BASE TABLE", "user", "varchar"},
	{"test", "comments", "BASE TABLE", "created_at", "timestamp"},
}

var indexes = map[string][]string{
	"pulls":    {"index_pull_number"},
	"comments": {"index_comments_pull_number"},
}
