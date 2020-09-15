# go-sqlsmith
[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2Fchaos-mesh%2Fgo-sqlsmith.svg?type=shield)](https://app.fossa.com/projects/git%2Bgithub.com%2Fchaos-mesh%2Fgo-sqlsmith?ref=badge_shield)


Go version of [SQLsmith](https://github.com/anse1/sqlsmith).

## Usage

```go
import (
	sqlsmith_go "github.com/chaos-mesh/go-sqlsmith"
)

func gosmith() {
	ss := sqlsmith_go.New()

	// load schema
	ss.LoadSchema([][5]string{
		// members table
		[5]string{"games", "members", "BASE TABLE", "id", "int(11)"},
		[5]string{"games", "members", "BASE TABLE", "name", "varchar(255)"},
		[5]string{"games", "members", "BASE TABLE", "age", "int(11)"},
		[5]string{"games", "members", "BASE TABLE", "team_id", "int(11)"},
		// teams table
		[5]string{"games", "teams", "BASE TABLE", "id", "int(11)"},
		[5]string{"games", "teams", "BASE TABLE", "team_name", "varchar(255)"},
		[5]string{"games", "teams", "BASE TABLE", "created_at", "timestamp"},
	})

	// use games database
	ss.SetDB("games")

	// generate select statement AST without scema information
	node := ss.SelectStmt(5)

	// fill the tree with selected schema and get SQL string
	sql, err := ss.Walk(node)
}
```


## License
[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2Fchaos-mesh%2Fgo-sqlsmith.svg?type=large)](https://app.fossa.com/projects/git%2Bgithub.com%2Fchaos-mesh%2Fgo-sqlsmith?ref=badge_large)