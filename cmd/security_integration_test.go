package main

import (
	"context"
	"database/sql"
	"os"
	"strconv"
	"sync"
	"testing"

	sqlsmith "github.com/chaos-mesh/go-sqlsmith"
	"github.com/chaos-mesh/go-sqlsmith/internal/dbutil"
	"github.com/chaos-mesh/go-sqlsmith/util"
	"github.com/pingcap/tidb/pkg/parser/ast"
	"github.com/stretchr/testify/require"
)

func integrationOptions(t *testing.T) dbutil.Options {
	t.Helper()
	name := os.Getenv("SQLSMITH_TEST_DB")
	if name == "" {
		t.Skip("set SQLSMITH_TEST_DB to a disposable database with a schema-scoped account")
	}
	port := 3306
	if value := os.Getenv("SQLSMITH_TEST_PORT"); value != "" {
		var err error
		port, err = strconv.Atoi(value)
		require.NoError(t, err)
	}
	return dbutil.Options{Host: "127.0.0.1", Port: port, User: "sqlsmith", Database: name,
		Socket: os.Getenv("SQLSMITH_TEST_SOCKET"), PasswordFile: os.Getenv("SQLSMITH_TEST_PASSWORD_FILE")}
}

func liveConnect(t *testing.T, config *SQLSmithConfig) *Connect {
	t.Helper()
	opts := integrationOptions(t)
	_, cfg, err := dbutil.Configs(opts)
	require.NoError(t, err)
	c, err := NewConnect(cfg, opts.Database, config)
	require.NoError(t, err)
	require.NoError(t, c.db.Ping())
	t.Cleanup(func() {
		c.db.Close()
		for _, exec := range c.dbs {
			exec.db.Close()
		}
	})
	return c
}

func TestLiveQuotedMetadataAndInsertBuilders(t *testing.T) {
	c := liveConnect(t, NewSQLSmithConfig())
	table, column, index := "metadata` JOIN mysql_user ON 1=1 #", "value` /*data*/", "idx*/ WHERE 1=1 #"
	quoted := util.QuoteIdentifier(table)
	_, err := c.db.Exec("CREATE TABLE " + quoted + " (id INT PRIMARY KEY AUTO_INCREMENT, " + util.QuoteIdentifier(column) + " VARCHAR(255), KEY " + util.QuoteIdentifier(index) + " (" + util.QuoteIdentifier(column) + "))")
	require.NoError(t, err)
	t.Cleanup(func() { c.db.Exec("DROP TABLE IF EXISTS " + quoted) })
	columns, err := c.FetchColumns(c.dbname, table)
	require.NoError(t, err)
	require.Len(t, columns, 2)
	require.Equal(t, column, columns[1][0])
	indexes, err := c.FetchIndexes(c.dbname, table)
	require.NoError(t, err)
	require.Contains(t, indexes, index)
	schema, err := c.FetchSchema(c.dbname)
	require.NoError(t, err)
	var ownSchema [][5]string
	for _, row := range schema {
		if row[1] == table {
			ownSchema = append(ownSchema, row)
		}
	}
	ss := sqlsmith.New()
	ss.SetDB(c.dbname)
	ss.SetHint(true)
	ss.LoadSchema(ownSchema, map[string][]string{table: indexes})
	sql, _, err := ss.InsertStmtStr(false)
	require.NoError(t, err)
	_, err = c.db.Exec(sql)
	require.NoError(t, err, sql)
	sql, _, err = ss.InsertStmt(false)
	require.NoError(t, err)
	_, err = c.db.Exec(sql)
	require.NoError(t, err, sql)
	batch, err := ss.BatchData(3, 2)
	require.NoError(t, err)
	for _, sql := range batch {
		_, err = c.db.Exec(sql)
		require.NoError(t, err, sql)
	}
	generator, err := ss.GenData(3, 2)
	require.NoError(t, err)
	for batch := generator.Next(); len(batch) != 0; batch = generator.Next() {
		for _, sql := range batch {
			_, err = c.db.Exec(sql)
			require.NoError(t, err, sql)
		}
	}
	for i := 0; i < 20; i++ {
		selectTree := &ast.SelectStmt{
			SelectStmtOpts: &ast.SelectStmtOpts{}, Fields: &ast.FieldList{},
			From:       &ast.TableRefsClause{TableRefs: &ast.Join{Left: &ast.TableName{}}},
			TableHints: []*ast.TableOptimizerHint{{}, {}},
		}
		sql, _, err = ss.Walk(selectTree)
		require.NoError(t, err)
		rows, err := c.db.Query(sql)
		require.NoError(t, err, sql)
		require.NoError(t, rows.Close())
	}
	drop := &ast.AlterTableStmt{Table: &ast.TableName{Name: ast.NewCIStr(table)}, Specs: []*ast.AlterTableSpec{{Tp: ast.AlterTableDropIndex, Name: index}}}
	sql, err = util.BufferOut(drop)
	require.NoError(t, err)
	_, err = c.db.Exec(sql)
	require.NoError(t, err, sql)
	columns, err = c.FetchColumns(c.dbname, table)
	require.NoError(t, err)
	require.Len(t, columns, 2)
	for i := 0; i < 10; i++ {
		_, err = c.FetchColumns(c.dbname, "missing` table")
		require.Error(t, err)
		_, err = c.FetchIndexes(c.dbname, table)
		require.NoError(t, err)
	}
	require.Zero(t, c.db.Stats().InUse)
}

func TestLiveSessionModeSurvivesPoolAndReconnect(t *testing.T) {
	c := liveConnect(t, NewSQLSmithConfig())
	var before string
	require.NoError(t, c.db.QueryRow("SELECT @@GLOBAL.sql_mode").Scan(&before))
	check := func(db *sql.DB) {
		t.Helper()
		first, err := db.Conn(context.Background())
		require.NoError(t, err)
		var mode string
		require.NoError(t, first.QueryRowContext(context.Background(), "SELECT @@SESSION.sql_mode").Scan(&mode))
		require.Equal(t, "NO_ENGINE_SUBSTITUTION", mode)
		if db.Stats().MaxOpenConnections > 1 {
			second, err := db.Conn(context.Background())
			require.NoError(t, err)
			require.NoError(t, second.QueryRowContext(context.Background(), "SELECT @@SESSION.sql_mode").Scan(&mode))
			require.Equal(t, "NO_ENGINE_SUBSTITUTION", mode)
			require.NoError(t, second.Close())
		}
		require.NoError(t, first.Close())
	}
	check(c.db)
	require.NoError(t, c.ReConnect(c.dbConfig))
	check(c.db)
	require.NoError(t, c.forkDB(2))
	for _, exec := range c.dbs {
		check(exec.db)
	}
	var after string
	require.NoError(t, c.db.QueryRow("SELECT @@GLOBAL.sql_mode").Scan(&after))
	require.Equal(t, before, after)
}

func TestLiveConcurrentGenerationAndReload(t *testing.T) {
	c := liveConnect(t, &SQLSmithConfig{DMLInsert: 1})
	_, err := c.db.Exec("CREATE TABLE concurrent_fixture (id INT PRIMARY KEY AUTO_INCREMENT, value VARCHAR(255))")
	require.NoError(t, err)
	t.Cleanup(func() { c.db.Exec("DROP TABLE IF EXISTS concurrent_fixture") })
	require.NoError(t, c.ReloadSchema())
	// Keep this test's writes on its own fixture even if the disposable
	// schema already contains tables from the integration binary.
	c.ss.Databases[c.dbname].Online = true
	c.ss.Databases[c.dbname].Tables["concurrent_fixture"].Online = true
	require.NoError(t, c.forkDB(4))
	var wg sync.WaitGroup
	failures := make(chan error, 100)
	for worker := 0; worker < 4; worker++ {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()
			for i := 0; i < 20; i++ {
				if _, err := c.RunOnce(worker); err != nil {
					failures <- err
				}
			}
		}(worker)
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 20; i++ {
			if err := c.ReloadSchema(); err != nil {
				failures <- err
			}
		}
	}()
	wg.Wait()
	close(failures)
	for err := range failures {
		require.NoError(t, err)
	}
}

func TestLiveTLSVerification(t *testing.T) {
	opts := integrationOptions(t)
	opts.TLSCA = os.Getenv("SQLSMITH_TEST_TLS_CA")
	if opts.TLSCA == "" {
		t.Skip("set SQLSMITH_TEST_TLS_CA to the disposable server CA")
	}
	opts.TLS = true
	opts.Socket = ""
	_, cfg, err := dbutil.Configs(opts)
	require.NoError(t, err)
	db, err := dbutil.Open(cfg)
	require.NoError(t, err)
	defer db.Close()
	require.NoError(t, db.Ping())
	var name, cipher string
	require.NoError(t, db.QueryRow("SHOW STATUS LIKE 'Ssl_cipher'").Scan(&name, &cipher))
	require.NotEmpty(t, cipher)
	opts.TLSServerName = "wrong-host.invalid"
	_, cfg, err = dbutil.Configs(opts)
	require.NoError(t, err)
	wrong, err := dbutil.Open(cfg)
	require.NoError(t, err)
	defer wrong.Close()
	require.Error(t, wrong.Ping())
	opts.TLSServerName, opts.TLSCA = "", ""
	_, cfg, err = dbutil.Configs(opts)
	require.NoError(t, err)
	untrusted, err := dbutil.Open(cfg)
	require.NoError(t, err)
	defer untrusted.Close()
	require.Error(t, untrusted.Ping())
}
