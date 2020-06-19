module github.com/chaos-mesh/go-sqlsmith

go 1.13

require (
	github.com/BurntSushi/toml v0.3.1
	github.com/aws/aws-sdk-go v1.32.4
	github.com/go-sql-driver/mysql v1.5.0
	github.com/ideahitme/segment v0.0.0-20170608010804-7338e16e3974
	github.com/juju/errors v0.0.0-20190930114154-d42613fe1ab9
	github.com/juju/testing v0.0.0-20200608005635-e4eedbc6f7aa // indirect
	github.com/ngaut/log v0.0.0-20180314031856-b8e36e7ba5ac
	github.com/pingcap/parser v0.0.0-20200317021010-cd90cc2a7d87
	github.com/pingcap/tidb v2.1.0-beta+incompatible
	github.com/satori/go.uuid v1.2.0
	github.com/stretchr/testify v1.5.1
	go.etcd.io/etcd v0.5.0-alpha.5.0.20191023171146-3cf2f69b5738
)

replace google.golang.org/grpc => google.golang.org/grpc v1.26.0

// we use pingcap/pd and pingcap/pd/v4 at the same time, which will cause a panic because pd register prometheus metrics two times.
replace github.com/pingcap/pd => github.com/mahjonp/pd v1.1.0-beta.0.20200408110858-9c088a87390c

replace github.com/pingcap/tidb => github.com/pingcap/tidb v0.0.0-20200317142013-5268094afe05

replace github.com/uber-go/atomic => go.uber.org/atomic v1.5.0
