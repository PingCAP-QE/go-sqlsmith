// Copyright 2020 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package builtin

import (
	"log"
	"math/rand"

	"github.com/pingcap/tidb/pkg/parser/ast"

	"github.com/chaos-mesh/go-sqlsmith/types"
	"github.com/chaos-mesh/go-sqlsmith/util"
)

type hintClass struct {
	name     string
	minArg   int
	maxArg   int
	constArg bool
	mysql    bool
	stable   bool
}

var hintKeywords = []*hintClass{
	// with no args
	{"hash_agg", 0, 0, false, false, false},
	{"stream_agg", 0, 0, false, false, false},
	{"agg_to_cop", 0, 0, false, false, false},
	{"read_consistent_replica", 0, 0, false, false, false},
	{"no_index_merge", 0, 0, false, false, false},

	// with bool (TRUE or FALSE)
	{"use_toja", 1, 1, false, false, false},
	{"enable_plan_cache", 1, 1, false, false, false},
	{"use_cascades", 1, 1, false, false, false},

	// these have been renamed
	// {"tidb_hj", 2, 3, false, false, true},
	// {"tidb_smj", 2, 3, false, false, true},
	// {"tidb_inlj", 2, 3, false, false, true},
	// with 2 or more args
	{"hash_join", 1, -1, false, true, false},
	{"merge_join", 1, -1, false, false, false},
	{"inl_join", 1, -1, false, false, false},

	// with int (byte)
	{"memory_quota", 1, 1, false, false, false},
	// with int (ms)
	{"max_execution_time", 1, 1, false, false, false},
}

var indexHintKeywords = []*hintClass{
	// with table name and at least one idx name
	{"use_index", 2, -1, false, false, false},
	{"ignore_index", 2, -1, false, false, false},
	{"use_index_merge", 2, -1, false, false, false},
}

// these will not be generated for some reason
var disabledHintKeywords = []*hintClass{
	{"qb_name", 0, 0, false, false, false},

	// not released?
	{"time_range", 2, -1, false, false, false},
	// storage type with tablename: TIKV[t1]
	{"read_from_storage", 2, -1, false, false, false},
	// not released?
	{"query_type", 1, 1, false, false, false},

	{"inl_hash_join", 1, -1, false, false, false},
	{"inl_merge_join", 1, -1, false, false, false},
}

// GenerateHintExpr ...
func GenerateHintExpr(table *types.Table) (h *ast.TableOptimizerHint) {
	enabledKeywords := hintKeywords
	if len(table.Indexes) > 0 {
		enabledKeywords = append(enabledKeywords, indexHintKeywords...)
	}
	h = new(ast.TableOptimizerHint)
	hintKeyword := enabledKeywords[util.Rd(len(enabledKeywords))]
	h.HintName = ast.NewCIStr(hintKeyword.name)

	if hintKeyword.maxArg == 0 {
		return
	}

	if hintKeyword.maxArg == 1 {
		switch hintKeyword.name {
		case "use_toja", "enable_plan_cache", "use_cascades":
			h.HintData = util.RdBool()
		case "memory_quota":
			h.HintData = int64(util.RdRange(30720000, 40960000))
		case "max_execution_time":
			h.HintData = uint64(util.RdRange(500, 1500))
		default:
			log.Fatalf("unreachable hintKeyword.name:%s", hintKeyword.name)
		}
		return
	}

	shuffledTables := make([]ast.HintTable, 0)
	for _, t := range table.InnerTableList {
		shuffledTables = append(shuffledTables, ast.HintTable{
			TableName: ast.NewCIStr(t.Table),
		})
	}
	rand.Shuffle(len(shuffledTables), func(i, j int) {
		shuffledTables[i], shuffledTables[j] = shuffledTables[j], shuffledTables[i]
	})

	shuffledIndexes := make([]ast.CIStr, 0)
	for _, idx := range table.Indexes {
		if idx != "" {
			shuffledIndexes = append(shuffledIndexes, ast.NewCIStr(idx))
		}
	}
	rand.Shuffle(len(shuffledIndexes), func(i, j int) {
		shuffledIndexes[i], shuffledIndexes[j] = shuffledIndexes[j], shuffledIndexes[i]
	})

	switch hintKeyword.name {
	case "hash_join", "merge_join", "inl_join", "inl_hash_join", "inl_merge_join":
		if len(shuffledTables) < 2 {
			h = nil
			return
		}

		n := util.MinInt(util.Rd(4)+2, len(shuffledTables)) // avoid case n < 2
		for ; n > 0; n-- {
			h.Tables = append(h.Tables, shuffledTables[n-1])
		}
	case "use_index", "ignore_index", "use_index_merge":
		// if no table nor index return empty
		if len(shuffledTables) == 0 || len(shuffledIndexes) == 0 {
			h = nil
			return
		}
		h.Tables = append(h.Tables, shuffledTables[util.Rd(len(shuffledTables))])
		n := util.MinInt(util.Rd(4)+1, len(shuffledIndexes)) // avoid case n == 0
		for ; n > 0; n-- {
			h.Indexes = append(h.Indexes, shuffledIndexes[n-1])
		}
	default:
		log.Fatalf("unreachable hintKeyword.name:%s", hintKeyword.name)
	}
	return
}
