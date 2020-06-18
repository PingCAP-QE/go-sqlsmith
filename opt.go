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

package sqlsmith

// DMLOptions for DML generation
type DMLOptions struct {
	// if OnlineTable is set to true
	// generator will rand some tables from the target database
	// and the following DML statements before transaction closed
	// should only affects these tables
	// else there will be no limit for affected tables
	// if you OnlineDDL field in DDLOptions is true, you may got the following error from TiDB
	// "ERROR 1105 (HY000): Information schema is changed. [try again later]"
	OnlineTable bool
}

// DDLOptions for DDL generation
type DDLOptions struct {
	// if OnlineDDL is set to false
	// DDL generation will look up tables other generators which are doing DMLs with the tables
	// the DDL generated will avoid modifing these tables
	// if OnlineDDL set to true
	// DDL generation will not avoid online tables
	OnlineDDL bool
	// if OnlineDDL is set to false
	// Tables contains all online tables which should not be modified with DDL
	// pocket will collect them from other generator instances
	Tables []string
}
