module github.com/qiniu/bearer-token-service/v2

go 1.21.12

toolchain go1.22.0

require (
	github.com/DATA-DOG/go-sqlmock v1.5.2
	github.com/bradfitz/gomemcache v0.0.0-20250403215159-8d39553ac7cf
	github.com/go-sql-driver/mysql v1.9.3
	github.com/gorilla/mux v1.8.1
	github.com/prometheus/client_golang v1.18.0
	github.com/qiniu/bytes v0.0.0-00010101000000-000000000000
	github.com/qiniu/go-sdk/v7 v7.25.6
	github.com/qiniu/xlog.v1 v0.0.0-00010101000000-000000000000
	github.com/redis/go-redis/v9 v9.5.0
	github.com/stretchr/testify v1.11.1
	go.mongodb.org/mongo-driver v1.13.1
	golang.org/x/sync v0.5.0
	qiniu.com/auth/digest v0.0.0-00010101000000-000000000000
	qiniu.com/auth/proto.v1 v0.0.0-00010101000000-000000000000
)

replace (
	github.com/qiniu/bytes => ./pkg/bytes
	github.com/qiniu/bytes/seekable => ./pkg/bytes/seekable
	github.com/qiniu/log.v1 => ./pkg/log.v1
	github.com/qiniu/xlog.v1 => ./pkg/xlog.v1
	qiniu.com/auth/digest => ./pkg/auth/digest
	qiniu.com/auth/proto.v1 => ./pkg/auth/proto.v1
)

require (
	filippo.io/edwards25519 v1.1.0 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/klauspost/compress v1.17.4 // indirect
	github.com/matttproud/golang_protobuf_extensions/v2 v2.0.0 // indirect
	github.com/montanaflynn/stats v0.7.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/prometheus/client_model v0.5.0 // indirect
	github.com/prometheus/common v0.45.0 // indirect
	github.com/prometheus/procfs v0.12.0 // indirect
	github.com/qiniu/log.v1 v0.0.0-00010101000000-000000000000 // indirect
	github.com/stretchr/objx v0.5.2 // indirect
	github.com/xdg-go/pbkdf2 v1.0.0 // indirect
	github.com/xdg-go/scram v1.1.2 // indirect
	github.com/xdg-go/stringprep v1.0.4 // indirect
	github.com/youmark/pkcs8 v0.0.0-20201027041543-1326539a0a0a // indirect
	golang.org/x/crypto v0.17.0 // indirect
	golang.org/x/sys v0.15.0 // indirect
	golang.org/x/text v0.14.0 // indirect
	google.golang.org/protobuf v1.31.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
