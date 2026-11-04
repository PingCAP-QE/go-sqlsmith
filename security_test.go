package sqlsmith

import (
	"testing"

	"github.com/pingcap/tidb/pkg/parser"
	"github.com/pingcap/tidb/pkg/parser/ast"
	"github.com/stretchr/testify/require"
)

type collectTables struct{ names []string }

func (v *collectTables) Enter(n ast.Node) (ast.Node, bool) {
	if table, ok := n.(*ast.TableName); ok {
		v.names = append(v.names, table.Name.O)
	}
	return n, false
}
func (v *collectTables) Leave(n ast.Node) (ast.Node, bool) { return n, true }

func TestStringInsertBuildersPreserveIdentifiers(t *testing.T) {
	table, column := "t` VALUES (1) #", "c`x"
	ss := New()
	ss.SetDB("test")
	ss.LoadSchema([][5]string{{"test", table, "BASE TABLE", column, "varchar"}}, nil)
	check := func(sql string) {
		parsed, err := parser.New().ParseOneStmt(sql, "", "")
		require.NoError(t, err, sql)
		insert := parsed.(*ast.InsertStmt)
		require.Equal(t, table, insert.Table.TableRefs.Left.(*ast.TableSource).Source.(*ast.TableName).Name.O)
		require.Len(t, insert.Columns, 1)
		require.Equal(t, column, insert.Columns[0].Name.O)
	}
	sql, _, err := ss.InsertStmtStr(false)
	require.NoError(t, err)
	check(sql)
	batch, err := ss.BatchData(3, 2)
	require.NoError(t, err)
	require.Len(t, batch, 2)
	for _, sql := range batch {
		check(sql)
	}
	gen, err := ss.GenData(3, 2)
	require.NoError(t, err)
	for batch := gen.Next(); len(batch) > 0; batch = gen.Next() {
		for _, sql := range batch {
			check(sql)
		}
	}
}

func TestReservationsCoverNestedSelectsAndDDL(t *testing.T) {
	ss := New()
	ss.SetDB("test")
	ss.LoadSchema([][5]string{
		{"test", "allowed", "BASE TABLE", "c", "int"},
		{"test", "excluded", "BASE TABLE", "c", "int"},
	}, nil)
	db := ss.Databases["test"]
	db.Online, db.Tables["allowed"].Online = true, true
	for i := 0; i < 30; i++ {

		for _, tree := range []ast.Node{ss.selectStmt(3), ss.selectForUpdateStmt(3)} {
			_, _, err := ss.Walk(tree)
			require.NoError(t, err)
			visitor := &collectTables{}
			tree.Accept(visitor)
			require.NotEmpty(t, visitor.names)
			for _, name := range visitor.names {
				require.Equal(t, "allowed", name)
			}
		}
	}
	batch, err := ss.BatchData(1, 1)
	require.NoError(t, err)
	require.Len(t, batch, 1)
	insert, err := parser.New().ParseOneStmt(batch[0], "", "")
	require.NoError(t, err)
	require.Equal(t, "allowed", insert.(*ast.InsertStmt).Table.TableRefs.Left.(*ast.TableSource).Source.(*ast.TableName).Name.O)
	sql, err := ss.CreateIndexStmt(&DDLOptions{Tables: []string{"excluded"}})
	require.NoError(t, err)
	parsed, err := parser.New().ParseOneStmt(sql, "", "")
	require.NoError(t, err)
	require.Equal(t, "allowed", parsed.(*ast.CreateIndexStmt).Table.Name.O)
	require.False(t, db.Tables["excluded"].OnlineOther)
	db.Tables["excluded"].OnlineOther = true
	_, err = ss.CreateIndexStmt(&DDLOptions{Tables: []string{"allowed"}})
	require.Error(t, err)
	require.True(t, db.Tables["excluded"].OnlineOther)
	require.False(t, db.Tables["allowed"].OnlineOther)
	db.Tables["allowed"].OnlineOther = true
	_, err = ss.CreateIndexStmt(&DDLOptions{OnlineDDL: true})
	require.NoError(t, err)
	require.True(t, db.Tables["allowed"].OnlineOther)
	require.True(t, db.Tables["excluded"].OnlineOther)
	db.Tables["allowed"].Online = false
	_, _, err = ss.SelectForUpdateStmt(1)
	require.Error(t, err)
}

func TestUnloadedSchemaReturnsErrors(t *testing.T) {
	ss := New()
	ss.SetDB("not_loaded")
	_, _, err := ss.SelectStmt(1)
	require.Error(t, err)
	_, _, err = ss.InsertStmtStr(false)
	require.Error(t, err)
	_, err = ss.AlterTableStmt(nil)
	require.Error(t, err)
	_, _, err = ss.CreateTableStmt()
	require.NoError(t, err)
}

func TestUpdateHasWritableAssignment(t *testing.T) {
	ss := New()
	ss.SetDB("test")
	ss.LoadSchema([][5]string{
		{"test", "one", "BASE TABLE", "id", "int"},
		{"test", "one", "BASE TABLE", "value", "varchar"},
	}, nil)
	for i := 0; i < 100; i++ {
		sql, _, err := ss.UpdateStmt()
		require.NoError(t, err)
		parsed, err := parser.New().ParseOneStmt(sql, "", "")
		require.NoError(t, err, sql)
		require.NotEmpty(t, parsed.(*ast.UpdateStmt).List)
	}
	delete(ss.Databases["test"].Tables["one"].Columns, "value")
	_, _, err := ss.UpdateStmt()
	require.Error(t, err)
}
