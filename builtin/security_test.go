package builtin

import (
	"testing"

	"github.com/chaos-mesh/go-sqlsmith/types"
	"github.com/chaos-mesh/go-sqlsmith/util"
	"github.com/pingcap/tidb/pkg/parser"
	_ "github.com/pingcap/tidb/pkg/types/parser_driver"
	"github.com/stretchr/testify/require"
)

func TestHintCommentTerminator(t *testing.T) {
	for _, table := range []*types.Table{
		{Table: "x*/WHERE 1=1/*"},
		{Table: "safe", InnerTableList: []*types.Table{{Table: "x*/WHERE 1=1/*"}}},
		{Table: "safe", Indexes: []string{"x*/WHERE 1=1/*"}},
	} {
		require.Nil(t, GenerateHintExpr(table))
	}
	safe := &types.Table{Table: "select", InnerTableList: []*types.Table{{Table: "has space"}}, Indexes: []string{"a`b"}}
	generated := 0
	for i := 0; i < 100; i++ {
		hint := GenerateHintExpr(safe)
		if hint == nil {
			continue
		}
		sql, err := util.BufferOut(hint)
		require.NoError(t, err)
		require.NotContains(t, sql, "*/")
		generated++
	}
	require.Positive(t, generated)
}

func TestStableFunctionsWithEmptyArityCatalog(t *testing.T) {
	for _, args := range []int{0, 1, 2, 3, 20} {
		expr := GenerateFuncCallExpr(nil, args, true)
		sql, err := util.BufferOut(expr)
		require.NoError(t, err)
		_, err = parser.New().ParseOneStmt("SELECT "+sql, "", "")
		require.NoError(t, err, sql)
	}
}
