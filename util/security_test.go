package util_test

import (
	"testing"

	"github.com/chaos-mesh/go-sqlsmith/util"
	"github.com/pingcap/tidb/pkg/parser"
	"github.com/pingcap/tidb/pkg/parser/ast"
	_ "github.com/pingcap/tidb/pkg/types/parser_driver"
	"github.com/stretchr/testify/require"
)

func TestRestoreIdentifiersRemainSingleNames(t *testing.T) {
	for _, name := range []string{"select", "has space", "中文", "a.b", "x` DROP COLUMN victim, ADD INDEX `y", "x*/WHERE 1=1/*"} {
		t.Run(name, func(t *testing.T) {
			node := &ast.AlterTableStmt{
				Table: &ast.TableName{Schema: ast.NewCIStr(name), Name: ast.NewCIStr(name)},
				Specs: []*ast.AlterTableSpec{{Tp: ast.AlterTableDropIndex, Name: name}},
			}
			sql, err := util.BufferOut(node)
			require.NoError(t, err)
			parsed, err := parser.New().ParseOneStmt(sql, "", "")
			require.NoError(t, err, sql)
			alter := parsed.(*ast.AlterTableStmt)
			require.Equal(t, name, alter.Table.Schema.O)
			require.Equal(t, name, alter.Table.Name.O)
			require.Len(t, alter.Specs, 1)
			require.Equal(t, name, alter.Specs[0].Name)
		})
	}
}

func TestRestoreStringRoundTrip(t *testing.T) {
	value := "quote\" and backslash\\ followed by \" OR 1=1 #"
	node := &ast.SelectStmt{SelectStmtOpts: &ast.SelectStmtOpts{}, Fields: &ast.FieldList{
		Fields: []*ast.SelectField{{Expr: ast.NewValueExpr(value, "", "")}},
	}}
	sql, err := util.BufferOut(node)
	require.NoError(t, err)
	parsed, err := parser.New().ParseOneStmt(sql, "", "")
	require.NoError(t, err, sql)
	require.Equal(t, value, parsed.(*ast.SelectStmt).Fields.Fields[0].Expr.(ast.ValueExpr).GetValue())
}
