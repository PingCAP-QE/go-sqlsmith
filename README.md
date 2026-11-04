# go-sqlsmith

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

## CLI connection and test safety

Use a database account scoped to a disposable test schema. The binaries no longer
need `SUPER` or change global `sql_mode`; the driver sets the existing generator
mode (`NO_ENGINE_SUBSTITUTION`) on each physical session, including worker and
reconnected sessions. Generated DDL and DML still run with the chosen account's
privileges.

Both binaries accept the existing `-host`, `-port`, `-user`, `-passwd`, `-socket`
and `-db` options. Prefer `SQLSMITH_PASSWORD` or `-password-file` to putting a
password in process arguments. A password file may end with LF or CRLF; other
characters are preserved. A nonempty `-passwd` takes precedence over the
environment; `-password-file` takes precedence over the environment and cannot
be combined with a nonempty `-passwd`. Connection logs omit the password.

Remote TCP connections require `-tls`, which verifies the server certificate
and hostname. Use `-tls-ca /path/to/ca.pem` for a private CA, and optionally
`-tls-server-name db.example` when the connection address differs from the
certificate name. There is no certificate-verification bypass. Local loopback
and Unix-socket connections continue to support plaintext. An isolated remote
test network without TLS requires explicit `-allow-insecure-transport`.
Passwordless disposable instances require explicit `-allow-empty-password`.
The existing `root` user default remains for local test compatibility; use
`-user sqlsmith` with a schema-scoped account for normal runs.

For example, an administrator can provision a dedicated schema and grant only
its privileges to an existing test user:

```sql
CREATE DATABASE sqlsmith_test;
GRANT SELECT, INSERT, UPDATE, DELETE, CREATE, DROP, ALTER, INDEX
  ON sqlsmith_test.* TO 'sqlsmith'@'<runner-host>';
```

Run the fuzzer using a mounted password file:

```sh
make
./bin/sqlsmith -host db.example -user sqlsmith -db sqlsmith_test \
  -password-file /run/secrets/mysql-password -tls -tls-ca /path/to/ca.pem
```

The integration binary replaces its selected schema and refuses to connect
unless `-clear` is supplied. It uses `-db` consistently instead of always
dropping the database named `test`. `make integration-test` no longer selects
the host MySQL socket implicitly; pass the intended disposable target:

```sh
make integration-test TEST_FLAGS='-host 127.0.0.1 -port 3306 -user sqlsmith -db sqlsmith_test -password-file /run/secrets/mysql-password -clear'
```

The same `-clear` opt-in remains available on `bin/sqlsmith`. Bootstrap
statements and successful worker DDL have timestamped audit context without
connection credentials.

## Security regression tests

`go test ./...` covers SQL identifier and literal round trips, comment-safe
optimizer hints, reservation boundaries, stable function generation and
connection options. To include real-database tests (also under `-race`), supply
a disposable schema already granted to the `sqlsmith` account:

```sh
SQLSMITH_TEST_DB=sqlsmith_test SQLSMITH_TEST_PORT=3306 \
SQLSMITH_TEST_PASSWORD_FILE=/run/secrets/mysql-password \
  go test -race ./...
```

`SQLSMITH_PASSWORD` can be used instead of `SQLSMITH_TEST_PASSWORD_FILE`.
`SQLSMITH_TEST_SOCKET` selects a local socket. `SQLSMITH_TEST_TLS_CA` additionally
enables the live TLS handshake tests over TCP: a trusted certificate succeeds,
while an untrusted CA or wrong hostname fails. CI exercises MySQL 5.7 and 8.4
with a generated password and a schema-scoped user.

The library's public SQL generation signatures remain unchanged. Identifiers
are quoted rather than restricted to ASCII: reserved words, spaces, Unicode
and embedded backticks remain usable. Optional optimizer hints are omitted
when a referenced identifier contains a comment terminator. Use one mutable
`SQLSmith` instance per goroutine, or synchronize external access, including
access to its exported schema maps. The CLI synchronizes its shared generator
while allowing database executions to proceed concurrently.
