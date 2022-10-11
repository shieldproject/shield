module github.com/shieldproject/shield

go 1.13

require (
	github.com/Azure/azure-sdk-for-go v44.0.0+incompatible
	github.com/Azure/go-autorest/autorest v0.11.0 // indirect
	github.com/Azure/go-autorest/autorest/to v0.4.0 // indirect
	github.com/ErikDubbelboer/gspt v0.0.0-20180711091504-e39e726e09cc
	github.com/armon/go-metrics v0.3.10 // indirect
	github.com/cloudfoundry-community/vaultkv v0.3.0
	github.com/coreos/bbolt v1.3.2 // indirect
	github.com/coreos/etcd v3.3.18+incompatible // indirect
	github.com/coreos/go-semver v0.2.0 // indirect
	github.com/coreos/go-systemd v0.0.0-20190719114852-fd7a80b32e1f // indirect
	github.com/coreos/pkg v0.0.0-20180928190104-399ea9e2e55f // indirect
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.1.0 // indirect
	github.com/dnaeon/go-vcr v1.0.1 // indirect
	github.com/fsouza/go-dockerclient v0.0.0-20151130162558-d750ee8aff39
	github.com/go-sql-driver/mysql v1.2.1-0.20160602001021-3654d25ec346
	github.com/goccy/go-json v0.9.11 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/groupcache v0.0.0-20190129154638-5b532d6fd5ef // indirect
	github.com/google/btree v1.0.0 // indirect
	github.com/google/go-cmp v0.5.9 // indirect
	github.com/google/go-github v0.0.0-20150605201353-af17a5fa8537
	github.com/google/go-querystring v1.1.0 // indirect
	github.com/google/uuid v1.1.2-0.20190416172445-c2e93f3ae59f // indirect
	github.com/gorilla/websocket v1.4.0
	github.com/grpc-ecosystem/go-grpc-middleware v1.0.0 // indirect
	github.com/grpc-ecosystem/go-grpc-prometheus v1.2.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway v1.9.0 // indirect
	github.com/hashicorp/consul/api v1.11.0
	github.com/hashicorp/go-hclog v0.14.1 // indirect
	github.com/hashicorp/go-immutable-radix v1.3.0 // indirect
	github.com/hashicorp/go-msgpack v0.5.5 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/hashicorp/go-sockaddr v1.0.2 // indirect
	github.com/hashicorp/go-uuid v1.0.2 // indirect
	github.com/hashicorp/golang-lru v0.5.4 // indirect
	github.com/hashicorp/serf v0.10.0 // indirect
	github.com/jhunt/go-ansi v0.0.0-20181127194324-5fd839f108b6
	github.com/jhunt/go-cli v0.0.0-20180120230054-44398e595118
	github.com/jhunt/go-envirotron v0.0.0-20191007155228-c8f2a184ad0f
	github.com/jhunt/go-log v0.0.0-20171024033145-ddc1e3b8ed30
	github.com/jhunt/go-querytron v0.0.0-20190121150331-d03b28210bbc
	github.com/jhunt/go-s3 v0.0.0-20190122180757-14f44ecac95f
	github.com/jhunt/go-snapshot v0.0.0-20171017043618-9ad8f5ee37a2 // indirect
	github.com/jhunt/go-table v0.0.0-20181127194439-fcc252a20f4c
	github.com/jmoiron/sqlx v0.0.0-20160615151803-bdae0c3219c3
	github.com/jonboulle/clockwork v0.1.0 // indirect
	github.com/konsorten/go-windows-terminal-sequences v1.0.2 // indirect
	github.com/kurin/blazer v0.5.1
	github.com/lestrrat-go/blackmagic v1.0.1 // indirect
	github.com/lestrrat-go/iter v1.0.2 // indirect
	github.com/lestrrat-go/jwx v1.2.25 // indirect
	github.com/lib/pq v1.1.1 // indirect
	github.com/mattn/go-isatty v0.0.12
	github.com/mattn/go-shellwords v1.0.0
	github.com/mattn/go-sqlite3 v1.1.1-0.20161028142218-86681de00ade
	github.com/matttproud/golang_protobuf_extensions v1.0.2-0.20181231171920-c182affec369 // indirect
	github.com/mitchellh/go-testing-interface v1.14.0 // indirect
	github.com/mitchellh/mapstructure v1.4.1-0.20210112042008-8ebf2d61a8b4 // indirect
	github.com/ncw/swift v1.0.48-0.20190410202254-753d2090bb62
	github.com/niemeyer/pretty v0.0.0-20200227124842-a10e7caefd8e // indirect
	github.com/okta/okta-jwt-verifier-golang v1.3.1
	github.com/onsi/ginkgo v1.13.0
	github.com/onsi/gomega v1.10.1
	github.com/patrickmn/go-cache v2.1.0+incompatible // indirect
	github.com/pborman/uuid v0.0.0-20150824212802-cccd189d45f7
	github.com/prometheus/client_golang v1.4.0
	github.com/satori/go.uuid v1.2.0 // indirect
	github.com/soheilhy/cmux v0.1.4 // indirect
	github.com/thanhpk/randstr v1.0.4
	github.com/tmc/grpc-websocket-proxy v0.0.0-20190109142713-0ad062ec5ee5 // indirect
	github.com/xiang90/probing v0.0.0-20190116061207-43a291ad63a2 // indirect
	go.etcd.io/bbolt v1.3.5 // indirect
	go.etcd.io/etcd v3.3.18+incompatible
	go.opencensus.io v0.22.0 // indirect
	go.uber.org/atomic v1.4.1-0.20190731194737-ef0d20d85b01 // indirect
	go.uber.org/multierr v1.1.1-0.20190429210458-bd075f90b08f // indirect
	go.uber.org/zap v1.10.1-0.20190709142728-9a9fa7d4b5f0 // indirect
	golang.org/x/crypto v0.0.0-20220926161630-eccd6366d1be
	golang.org/x/net v0.0.0-20211209124913-491a49abca63 // indirect
	golang.org/x/oauth2 v0.0.0-20190604053449-0f29369cfe45
	golang.org/x/term v0.0.0-20201210144234-2321bbc49cbf // indirect
	golang.org/x/time v0.0.0-20200630173020-3af7569d3a1e // indirect
	google.golang.org/api v0.9.0
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/grpc v1.25.1 // indirect
	gopkg.in/check.v1 v1.0.0-20200227125254-8fa46927fb4f // indirect
	gopkg.in/yaml.v2 v2.4.0
	gopkg.in/yaml.v3 v3.0.0 // indirect
	sigs.k8s.io/yaml v1.2.0 // indirect
)
