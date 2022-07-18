module github.com/ottogroup/penelope

require (
	cloud.google.com/go v0.102.0
	cloud.google.com/go/bigquery v1.32.0
	cloud.google.com/go/logging v1.4.2
	cloud.google.com/go/monitoring v1.5.0
	cloud.google.com/go/storage v1.22.1
	cloud.google.com/go/trace v1.2.0 // indirect
	contrib.go.opencensus.io/exporter/stackdriver v0.13.13
	github.com/aws/aws-sdk-go v1.44.27
	github.com/go-pg/pg/v10 v10.10.6
	github.com/golang-jwt/jwt/v4 v4.4.1
	github.com/golang/glog v1.0.0
	github.com/google/uuid v1.3.0
	github.com/gorilla/mux v1.8.0
	github.com/jarcoal/httpmock v1.2.0
	github.com/pkg/errors v0.9.1
	github.com/prometheus/prometheus v0.36.0 // indirect
	github.com/stretchr/testify v1.7.1
	github.com/vmihailenco/msgpack/v5 v5.3.5 // indirect
	go.opencensus.io v0.23.0
	golang.org/x/crypto v0.0.0-20220525230936-793ad666bf5e // indirect
	google.golang.org/api v0.87.0
	google.golang.org/genproto v0.0.0-20220624142145-8cd45d7dbd1f
	google.golang.org/grpc v1.47.0
	google.golang.org/protobuf v1.28.0
	gopkg.in/dc0d/tinykv.v4 v4.0.1
	gopkg.in/yaml.v2 v2.4.0
)

go 1.16
