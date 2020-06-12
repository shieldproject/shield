module github.com/shieldproject/shield

go 1.13

require (
	cloud.google.com/go v0.58.0 // indirect
	github.com/Azure/azure-sdk-for-go v8.1.1-beta+incompatible
	github.com/ErikDubbelboer/gspt v0.0.0-20180711091504-e39e726e09cc
	github.com/alexbrainman/sspi v0.0.0-20180613141037-e580b900e9f5 // indirect
	github.com/beorn7/perks v1.0.1
	github.com/cloudfoundry-community/vaultkv v0.0.0-20200311151509-343c0e6fc506
	github.com/coreos/etcd v3.3.18+incompatible // indirect
	github.com/coreos/go-systemd v0.0.0-20190719114852-fd7a80b32e1f
	github.com/coreos/pkg v0.0.0-20180928190104-399ea9e2e55f
	github.com/envoyproxy/go-control-plane v0.9.4 // indirect
	github.com/fsouza/go-dockerclient v0.0.0-20151130162558-d750ee8aff39
	github.com/go-sql-driver/mysql v1.2.1-0.20160602001021-3654d25ec346
	github.com/gogo/protobuf v1.2.2-0.20190730201129-28a6bbf47e48
	github.com/golang/mock v1.4.3 // indirect
	github.com/golang/protobuf v1.4.2
	github.com/google/go-github v0.0.0-20150605201353-af17a5fa8537
	github.com/google/go-querystring v0.0.0-20150414214848-547ef5ac9797
	github.com/google/uuid v1.1.2-0.20190416172445-c2e93f3ae59f
	github.com/gorilla/websocket v1.4.2
	github.com/hashicorp/consul v0.8.0
	github.com/hashicorp/go-cleanhttp v0.5.1 // indirect
	github.com/hashicorp/serf v0.9.0 // indirect
	github.com/jcmturner/gokrb5/v8 v8.2.0 // indirect
	github.com/jhunt/go-ansi v0.0.0-20181127194324-5fd839f108b6
	github.com/jhunt/go-cli v0.0.0-20180120230054-44398e595118
	github.com/jhunt/go-envirotron v0.0.0-20191007155228-c8f2a184ad0f
	github.com/jhunt/go-log v0.0.0-20171024033145-ddc1e3b8ed30
	github.com/jhunt/go-querytron v0.0.0-20190121150331-d03b28210bbc
	github.com/jhunt/go-s3 v0.0.0-20200530154331-7efb75fe8c97
	github.com/jhunt/go-table v0.0.0-20181127194439-fcc252a20f4c
	github.com/jhunt/shield-storage-gateway v0.0.0-20200521135823-ceb24ff0859d
	github.com/jhunt/ssg v1.0.3
	github.com/jmoiron/sqlx v0.0.0-20160615151803-bdae0c3219c3
	github.com/kurin/blazer v0.5.1
	github.com/lib/pq v1.7.0
	github.com/mattn/go-isatty v0.0.12
	github.com/mattn/go-shellwords v1.0.0
	github.com/mattn/go-sqlite3 v1.1.1-0.20161028142218-86681de00ade
	github.com/matttproud/golang_protobuf_extensions v1.0.2-0.20181231171920-c182affec369
	github.com/ncw/swift v1.0.48-0.20190410202254-753d2090bb62
	github.com/onsi/ginkgo v1.12.2
	github.com/onsi/gomega v1.10.1
	github.com/pborman/uuid v0.0.0-20150824212802-cccd189d45f7
	github.com/prometheus/client_golang v1.1.0
	github.com/prometheus/client_model v0.0.0-20190812154241-14fe0d1b01d4
	github.com/prometheus/common v0.6.1-0.20190730175846-637d7c34db12
	github.com/prometheus/procfs v0.0.4-0.20190731153504-5da962fa40f1
	github.com/starkandwayne/goutils v0.0.0-20170530161610-d28cacc19462
	github.com/starkandwayne/safe v1.1.1-0.20190430135104-923d720c7365
	github.com/tredoe/osutil v0.0.0-20161130133508-7d3ee1afa71c
	go.etcd.io/etcd v3.3.18+incompatible
	go.opencensus.io v0.22.3 // indirect
	go.uber.org/atomic v1.4.1-0.20190731194737-ef0d20d85b01
	go.uber.org/multierr v1.1.1-0.20190429210458-bd075f90b08f
	go.uber.org/zap v1.10.1-0.20190709142728-9a9fa7d4b5f0
	golang.org/x/crypto v0.0.0-20200604202706-70a84ac30bf9
	golang.org/x/exp v0.0.0-20200224162631-6cc2880d07d6 // indirect
	golang.org/x/lint v0.0.0-20200302205851-738671d3881b // indirect
	golang.org/x/net v0.0.0-20200602114024-627f9648deb9
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d
	golang.org/x/sync v0.0.0-20200317015054-43a5402ce75a // indirect
	golang.org/x/sys v0.0.0-20200610111108-226ff32320da
	golang.org/x/text v0.3.2
	golang.org/x/time v0.0.0-20191024005414-555d28b269f0 // indirect
	google.golang.org/api v0.26.0
	google.golang.org/cloud v0.0.0-20160324202040-eb47ba841d53
	google.golang.org/genproto v0.0.0-20200611194920-44ba362f84c1
	google.golang.org/grpc v1.29.1
	gopkg.in/jcmturner/aescts.v1 v1.0.1 // indirect
	gopkg.in/jcmturner/dnsutils.v1 v1.0.1 // indirect
	gopkg.in/jcmturner/goidentity.v3 v3.0.0 // indirect
	gopkg.in/jcmturner/gokrb5.v7 v7.5.0 // indirect
	gopkg.in/jcmturner/rpc.v1 v1.1.0 // indirect
	gopkg.in/yaml.v2 v2.3.0
)

replace go.etcd.io/etcd => go.etcd.io/etcd v0.0.0-20200520232829-54ba9589114f
