module git.coinninja.net/backend/thunderdome

go 1.12

require (
	cloud.google.com/go v0.45.1 // indirect
	git.coinninja.net/backend/blocc v1.1.8
	git.coinninja.net/backend/cnauth v0.1.1
	github.com/DataDog/datadog-go v2.2.0+incompatible
	github.com/Microsoft/go-winio v0.4.14 // indirect
	github.com/blendle/zapdriver v1.1.6
	github.com/btcsuite/btcd v0.0.0-20190824003749-130ea5bddde3
	github.com/btcsuite/btcutil v0.0.0-20190425235716-9e5f4b9a998d
	github.com/btcsuite/btcwallet v0.0.0-20190906013808-ae43a2a200e9 // indirect
	github.com/containerd/containerd v1.2.8 // indirect
	github.com/davecgh/go-spew v1.1.1
	github.com/dhui/dktest v0.3.1 // indirect
	github.com/docker/distribution v2.7.1+incompatible // indirect
	github.com/docker/go-units v0.4.0 // indirect
	github.com/elazarl/go-bindata-assetfs v1.0.0
	github.com/frankban/quicktest v1.4.2 // indirect
	github.com/go-chi/chi v4.0.2+incompatible
	github.com/go-chi/cors v1.0.0
	github.com/go-chi/render v1.0.1
	github.com/gogo/protobuf v1.3.0
	github.com/golang-migrate/migrate/v4 v4.6.1
	github.com/golang/protobuf v1.3.2
	github.com/google/wire v0.3.0
	github.com/gorilla/mux v1.7.3 // indirect
	github.com/grpc-ecosystem/go-grpc-middleware v1.0.0
	github.com/grpc-ecosystem/grpc-gateway v1.11.1
	github.com/jmoiron/sqlx v1.2.0
	github.com/juju/clock v0.0.0-20190205081909-9c5c9712527c // indirect
	github.com/juju/errors v0.0.0-20190806202954-0232dcc7464d // indirect
	github.com/juju/loggo v0.0.0-20190526231331-6e530bcce5d8 // indirect
	github.com/juju/testing v0.0.0-20190723135506-ce30eb24acd2 // indirect
	github.com/lib/pq v1.2.0
	github.com/lightningnetwork/lnd v0.7.1-beta.0.20190807225126-ea77ff91c221
	github.com/mattn/go-sqlite3 v1.11.0 // indirect
	github.com/rogpeppe/fastuuid v1.2.0 // indirect
	github.com/rs/xid v1.2.1
	github.com/snowzach/certtools v1.0.2
	github.com/spf13/cobra v0.0.5
	github.com/spf13/viper v1.4.0
	github.com/stretchr/testify v1.4.0
	go.opencensus.io v0.22.1
	go.uber.org/zap v1.10.0
	golang.org/x/crypto v0.0.0-20190907121410-71b5226ff739 // indirect
	golang.org/x/net v0.0.0-20190827160401-ba9fcec4b297
	golang.org/x/sys v0.0.0-20190907184412-d223b2b6db03 // indirect
	google.golang.org/api v0.10.0 // indirect
	google.golang.org/genproto v0.0.0-20190905072037-92dd089d5514
	google.golang.org/grpc v1.23.0
	gopkg.in/errgo.v1 v1.0.1 // indirect
	gopkg.in/macaroon-bakery.v2 v2.1.0 // indirect
	gopkg.in/macaroon.v2 v2.1.0
	gopkg.in/mgo.v2 v2.0.0-20190816093944-a6b53ec6cb22 // indirect
)

replace github.com/btcsuite/btcwallet v0.0.0-20180904010540-284e2e0e696e33d5be388f7f3d9a26db703e0c06 => github.com/btcsuite/btcwallet v0.0.0-20181130030754-284e2e0e696e

replace github.com/coreos/bbolt v0.0.0-20180223184059-7ee3ded59d4835e10f3e7d0f7603c42aa5e83820 => github.com/coreos/bbolt v1.3.1-etcd.8
