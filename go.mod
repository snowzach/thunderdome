module git.coinninja.net/backend/thunderdome

go 1.12

require (
	cloud.google.com/go v0.47.0 // indirect
	cloud.google.com/go/firestore v1.0.0 // indirect
	cloud.google.com/go/storage v1.1.1 // indirect
	git.coinninja.net/backend/blocc v1.1.15
	git.coinninja.net/backend/cnauth v0.1.2
	github.com/DataDog/datadog-go v2.3.0+incompatible
	github.com/Microsoft/go-winio v0.4.14 // indirect
	github.com/blendle/zapdriver v1.1.6
	github.com/btcsuite/btcd v0.20.0-beta
	github.com/btcsuite/btcutil v0.0.0-20190425235716-9e5f4b9a998d
	github.com/btcsuite/btcwallet v0.10.0 // indirect
	github.com/containerd/containerd v1.2.8 // indirect
	github.com/davecgh/go-spew v1.1.1
	github.com/dhui/dktest v0.3.1 // indirect
	github.com/docker/distribution v2.7.1+incompatible // indirect
	github.com/docker/go-units v0.4.0 // indirect
	github.com/elazarl/go-bindata-assetfs v1.0.0
	github.com/frankban/quicktest v1.4.2 // indirect
	github.com/gin-contrib/sse v0.1.0 // indirect
	github.com/gin-gonic/gin v1.4.0 // indirect
	github.com/go-chi/chi v4.0.2+incompatible
	github.com/go-chi/cors v1.0.0
	github.com/go-chi/render v1.0.1
	github.com/go-redis/redis v6.15.5+incompatible
	github.com/gobuffalo/pop v4.12.2+incompatible // indirect
	github.com/gogo/protobuf v1.3.1
	github.com/golang-migrate/migrate/v4 v4.6.2
	github.com/golang/groupcache v0.0.0-20191002201903-404acd9df4cc // indirect
	github.com/golang/protobuf v1.3.2
	github.com/google/wire v0.3.0
	github.com/gorilla/mux v1.7.3 // indirect
	github.com/grpc-ecosystem/go-grpc-middleware v1.1.0
	github.com/grpc-ecosystem/grpc-gateway v1.11.3
	github.com/jmoiron/sqlx v1.2.0
	github.com/jstemmer/go-junit-report v0.9.1 // indirect
	github.com/juju/clock v0.0.0-20190205081909-9c5c9712527c // indirect
	github.com/juju/errors v0.0.0-20190806202954-0232dcc7464d // indirect
	github.com/juju/loggo v0.0.0-20190526231331-6e530bcce5d8 // indirect
	github.com/juju/testing v0.0.0-20190723135506-ce30eb24acd2 // indirect
	github.com/lib/pq v1.2.0
	github.com/lightningnetwork/lnd v0.7.1-beta
	github.com/mattn/go-sqlite3 v1.11.0 // indirect
	github.com/pelletier/go-toml v1.5.0 // indirect
	github.com/rogpeppe/fastuuid v1.2.0 // indirect
	github.com/rs/xid v1.2.1
	github.com/snowzach/certtools v1.0.2
	github.com/spf13/cobra v0.0.5
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/spf13/viper v1.4.0
	github.com/stretchr/testify v1.4.0
	go.opencensus.io v0.22.1
	go.uber.org/multierr v1.2.0 // indirect
	go.uber.org/zap v1.10.0
	golang.org/x/crypto v0.0.0-20191011191535-87dc89f01550 // indirect
	golang.org/x/exp v0.0.0-20191014171548-69215a2ee97e // indirect
	golang.org/x/net v0.0.0-20191021124707-24d2ffbea1e8
	golang.org/x/sys v0.0.0-20191020212454-3e7259c5e7c2 // indirect
	golang.org/x/text v0.3.2
	golang.org/x/tools v0.0.0-20191018212557-ed542cd5b28a // indirect
	google.golang.org/api v0.11.0 // indirect
	google.golang.org/appengine v1.6.5 // indirect
	google.golang.org/genproto v0.0.0-20191009194640-548a555dbc03
	google.golang.org/grpc v1.24.0
	gopkg.in/errgo.v1 v1.0.1 // indirect
	gopkg.in/macaroon-bakery.v2 v2.1.0 // indirect
	gopkg.in/macaroon.v2 v2.1.0
	gopkg.in/mgo.v2 v2.0.0-20190816093944-a6b53ec6cb22 // indirect
)

replace github.com/btcsuite/btcwallet v0.0.0-20180904010540-284e2e0e696e33d5be388f7f3d9a26db703e0c06 => github.com/btcsuite/btcwallet v0.0.0-20181130030754-284e2e0e696e

replace github.com/coreos/bbolt v0.0.0-20180223184059-7ee3ded59d4835e10f3e7d0f7603c42aa5e83820 => github.com/coreos/bbolt v1.3.1-etcd.8
