module git.coinninja.net/backend/thunderdome

go 1.12

require (
	cloud.google.com/go v0.48.0 // indirect
	cloud.google.com/go/firestore v1.1.0 // indirect
	cloud.google.com/go/storage v1.3.0 // indirect
	firebase.google.com/go v3.10.0+incompatible // indirect
	git.coinninja.net/backend/blocc v1.1.16
	git.coinninja.net/backend/cnauth v0.1.2
	github.com/DataDog/datadog-go v3.2.0+incompatible
	github.com/Microsoft/go-winio v0.4.14 // indirect
	github.com/blendle/zapdriver v1.3.1
	github.com/btcsuite/btcd v0.20.1-beta
	github.com/btcsuite/btcutil v0.0.0-20190425235716-9e5f4b9a998d
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
	github.com/go-redis/redis v6.15.6+incompatible
	github.com/gogo/protobuf v1.3.1
	github.com/golang-migrate/migrate/v4 v4.7.0
	github.com/golang/groupcache v0.0.0-20191027212112-611e8accdfc9 // indirect
	github.com/golang/protobuf v1.3.2
	github.com/google/wire v0.3.0
	github.com/gorilla/mux v1.7.3 // indirect
	github.com/grpc-ecosystem/go-grpc-middleware v1.1.0
	github.com/grpc-ecosystem/grpc-gateway v1.12.1
	github.com/jmoiron/sqlx v1.2.0
	github.com/jstemmer/go-junit-report v0.9.1 // indirect
	github.com/lib/pq v1.2.0
	github.com/lightningnetwork/lnd v0.7.1-beta
	github.com/mattn/go-sqlite3 v1.11.0 // indirect
	github.com/pelletier/go-toml v1.6.0 // indirect
	github.com/rs/xid v1.2.1
	github.com/snowzach/certtools v1.0.2
	github.com/spf13/cobra v0.0.5
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/spf13/viper v1.5.0
	github.com/stretchr/testify v1.4.0
	go.opencensus.io v0.22.2 // indirect
	go.uber.org/multierr v1.4.0 // indirect
	go.uber.org/zap v1.13.0
	golang.org/x/crypto v0.0.0-20191117063200-497ca9f6d64f // indirect
	golang.org/x/net v0.0.0-20191116160921-f9c825593386
	golang.org/x/sys v0.0.0-20191118133127-cf1e2d577169 // indirect
	golang.org/x/text v0.3.2
	golang.org/x/tools v0.0.0-20191118051429-5a76f03bc7c3 // indirect
	google.golang.org/api v0.14.0 // indirect
	google.golang.org/appengine v1.6.5 // indirect
	google.golang.org/genproto v0.0.0-20191115221424-83cc0476cb11
	google.golang.org/grpc v1.25.1
	gopkg.in/macaroon-bakery.v2 v2.1.0 // indirect
	gopkg.in/macaroon.v2 v2.1.0
	gopkg.in/yaml.v2 v2.2.5 // indirect
)

replace github.com/btcsuite/btcwallet v0.0.0-20180904010540-284e2e0e696e33d5be388f7f3d9a26db703e0c06 => github.com/btcsuite/btcwallet v0.0.0-20181130030754-284e2e0e696e

replace github.com/coreos/bbolt v0.0.0-20180223184059-7ee3ded59d4835e10f3e7d0f7603c42aa5e83820 => github.com/coreos/bbolt v1.3.1-etcd.8
