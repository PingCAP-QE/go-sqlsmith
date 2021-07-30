GOARCH := $(if $(GOARCH),$(GOARCH),amd64)
GO=GO15VENDOREXPERIMENT="1" CGO_ENABLED=1 GOOS=$(GOOS) GOARCH=$(GOARCH) GO111MODULE=on go
GOTEST=GO15VENDOREXPERIMENT="1" CGO_ENABLED=1 GO111MODULE=on go test # go race detector requires cgo
VERSION   := $(if $(VERSION),$(VERSION),latest)
GOBUILD=$(GO) build -ldflags '$(LDFLAGS)'

sqlsmith:
	$(GOBUILD) $(GOMOD) -o bin/sqlsmith cmd/*.go

integration-test:
	$(GOBUILD) $(GOMOD) -o bin/test tests/*.go
	./bin/test -socket /var/run/mysqld/mysqld.sock

.PHONY: sqlsmith
