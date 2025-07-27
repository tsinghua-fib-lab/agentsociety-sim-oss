module git.fiblab.net/sim/simulet-go

go 1.23.0

toolchain go1.23.4

// replace git.fiblab.net/sim/protos/v2 => github.com/tsinghua-fib-lab/cityproto/v2 v2.2.10

require (
	connectrpc.com/connect v1.18.1
	git.fiblab.net/sim/protos/v2 v2.4.2
	git.fiblab.net/sim/routing/v2 v2.0.8
	git.fiblab.net/sim/syncer/v3 v3.2.1
	git.fiblab.net/utils/logrus-easy-formatter v0.1.0
	github.com/samber/lo v1.49.1
	github.com/sirupsen/logrus v1.9.3
	github.com/stretchr/testify v1.10.0
	go.mongodb.org/mongo-driver v1.17.3
	google.golang.org/protobuf v1.36.6
	gopkg.in/yaml.v2 v2.4.0
)

require (
	git.fiblab.net/general/common/v2 v2.6.3
	golang.org/x/exp v0.0.0-20250408133849-7e4ce0ab07d0
)

require (
	connectrpc.com/grpcreflect v1.3.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/golang/snappy v1.0.0 // indirect
	github.com/iancoleman/strcase v0.3.0 // indirect
	github.com/klauspost/compress v1.18.0 // indirect
	github.com/kr/pretty v0.3.0 // indirect
	github.com/montanaflynn/stats v0.7.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/puzpuzpuz/xsync/v3 v3.5.1 // indirect
	github.com/rogpeppe/go-internal v1.10.0 // indirect
	github.com/rs/cors v1.11.1 // indirect
	github.com/xdg-go/pbkdf2 v1.0.0 // indirect
	github.com/xdg-go/scram v1.1.2 // indirect
	github.com/xdg-go/stringprep v1.0.4 // indirect
	github.com/youmark/pkcs8 v0.0.0-20240726163527-a2c0da244d78 // indirect
	golang.org/x/crypto v0.37.0 // indirect
	golang.org/x/net v0.39.0 // indirect
	golang.org/x/sync v0.13.0 // indirect
	golang.org/x/sys v0.32.0 // indirect
	golang.org/x/text v0.24.0 // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
