module github.com/flowline-io/flowbot

go 1.24

require (
	code.gitea.io/sdk/gitea v0.21.0
	github.com/Masterminds/semver/v3 v3.3.1
	github.com/PuerkitoBio/goquery v1.10.3
	github.com/RealAlexandreAI/json-repair v0.0.14
	github.com/ThreeDotsLabs/watermill v1.4.6
	github.com/ThreeDotsLabs/watermill-redisstream v1.4.3
	github.com/VictoriaMetrics/metrics v1.37.0
	github.com/bsm/redislock v0.9.4
	github.com/bwmarrin/discordgo v0.29.0
	github.com/bytedance/sonic v1.13.2
	github.com/cenkalti/backoff v2.2.1+incompatible
	github.com/cloudwego/eino v0.3.38
	github.com/cloudwego/eino-ext/components/model/ollama v0.0.0-20250520101807-b2008771903a
	github.com/cloudwego/eino-ext/components/model/openai v0.0.0-20250520101807-b2008771903a
	github.com/creachadair/jrpc2 v1.3.1
	github.com/dgraph-io/ristretto/v2 v2.2.0
	github.com/docker/cli v28.2.2+incompatible
	github.com/docker/docker v28.2.2+incompatible
	github.com/docker/go-units v0.5.0
	github.com/dustin/go-humanize v1.0.1
	github.com/emicklei/go-restful/v3 v3.12.2
	github.com/expr-lang/expr v1.17.4
	github.com/flc1125/go-cron/v4 v4.5.5
	github.com/flowline-io/contrib/fiberzerolog v0.0.0-20250529160935-d348985506ed
	github.com/flowline-io/fiberswagger v0.0.0-20250529155353-3b5b94dbd01c
	github.com/fsnotify/fsnotify v1.9.0
	github.com/gen2brain/beeep v0.0.0-20240516210008-9c006672e7f4
	github.com/go-echarts/go-echarts/v2 v2.5.4
	github.com/go-playground/validator/v10 v10.26.0
	github.com/go-sql-driver/mysql v1.9.2
	github.com/goccy/go-json v0.10.5
	github.com/goccy/go-yaml v1.18.0
	github.com/gofiber/fiber/v3 v3.0.0-beta.4
	github.com/golang-migrate/migrate/v4 v4.18.3
	github.com/google/go-github/v72 v72.0.0
	github.com/google/uuid v1.6.0
	github.com/heimdalr/dag v1.5.0
	github.com/hekmon/transmissionrpc/v3 v3.0.0
	github.com/hibiken/asynq v0.25.1
	github.com/influxdata/cron v0.0.0-20201006132531-4bb0a200dcbe
	github.com/jmoiron/sqlx v1.4.0
	github.com/json-iterator/go v1.1.12
	github.com/lithammer/shortuuid/v4 v4.2.0
	github.com/looplab/fsm v1.0.3
	github.com/mark3labs/mcp-go v0.31.0
	github.com/maxence-charriere/go-app/v10 v10.1.3
	github.com/meilisearch/meilisearch-go v0.32.0
	github.com/minio/minio-go/v7 v7.0.92
	github.com/minio/selfupdate v0.6.0
	github.com/mmcdole/gofeed v1.3.0
	github.com/montanaflynn/stats v0.7.1
	github.com/pkoukk/tiktoken-go v0.1.7
	github.com/prometheus/client_model v0.6.2
	github.com/prometheus/common v0.64.0
	github.com/redis/go-redis/v9 v9.9.0
	github.com/rs/zerolog v1.34.0
	github.com/samber/lo v1.50.0
	github.com/samber/oops v1.18.1
	github.com/schollz/progressbar/v3 v3.18.0
	github.com/shirou/gopsutil/v4 v4.25.5
	github.com/slack-go/slack v0.17.0
	github.com/spf13/pflag v1.0.6
	github.com/spf13/viper v1.20.1
	github.com/stretchr/testify v1.10.0
	github.com/swaggo/swag v1.16.4
	github.com/tidwall/gjson v1.18.0
	github.com/urfave/cli/v3 v3.3.3
	github.com/valyala/fasthttp v1.62.0
	go.uber.org/automaxprocs v1.6.0
	go.uber.org/fx v1.24.0
	golang.org/x/crypto v0.38.0
	golang.org/x/exp v0.0.0-20250218142911-aa4b98e5adaa
	golang.org/x/sys v0.33.0
	golang.org/x/time v0.11.0
	golang.org/x/xerrors v0.0.0-20231012003039-104605ab7028
	gorm.io/driver/mysql v1.5.7
	gorm.io/gen v0.3.27
	gorm.io/gorm v1.30.0
	gorm.io/plugin/dbresolver v1.6.0
	gotest.tools/v3 v3.5.2
	miniflux.app/v2 v2.2.8
	resty.dev/v3 v3.0.0-beta.3
)

require (
	aead.dev/minisign v0.2.0 // indirect
	filippo.io/edwards25519 v1.1.0 // indirect
	github.com/42wim/httpsig v1.2.2 // indirect
	github.com/KyleBanks/depth v1.2.1 // indirect
	github.com/Microsoft/go-winio v0.6.2 // indirect
	github.com/Rican7/retry v0.3.1 // indirect
	github.com/andybalholm/brotli v1.1.1 // indirect
	github.com/andybalholm/cascadia v1.3.3 // indirect
	github.com/bytedance/sonic/loader v0.2.4 // indirect
	github.com/cenkalti/backoff/v3 v3.2.2 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/cloudwego/base64x v0.1.5 // indirect
	github.com/cloudwego/eino-ext/libs/acl/openai v0.0.0-20250519084852-38fafa73d9ea // indirect
	github.com/containerd/errdefs v1.0.0 // indirect
	github.com/containerd/errdefs/pkg v0.3.0 // indirect
	github.com/containerd/log v0.1.0 // indirect
	github.com/creachadair/mds v0.23.0 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/davidmz/go-pageant v1.0.2 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/distribution/reference v0.6.0 // indirect
	github.com/dlclark/regexp2 v1.11.4 // indirect
	github.com/docker/go-connections v0.5.0 // indirect
	github.com/ebitengine/purego v0.8.4 // indirect
	github.com/emirpasic/gods v1.18.1 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/fxamacker/cbor/v2 v2.8.0 // indirect
	github.com/gabriel-vasile/mimetype v1.4.8 // indirect
	github.com/getkin/kin-openapi v0.118.0 // indirect
	github.com/go-fed/httpsig v1.1.0 // indirect
	github.com/go-ini/ini v1.67.0 // indirect
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-ole/go-ole v1.2.6 // indirect
	github.com/go-openapi/jsonpointer v0.21.0 // indirect
	github.com/go-openapi/jsonreference v0.20.0 // indirect
	github.com/go-openapi/spec v0.20.11 // indirect
	github.com/go-openapi/swag v0.23.0 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/go-toast/toast v0.0.0-20190211030409-01e6764cf0a4 // indirect
	github.com/go-viper/mapstructure/v2 v2.2.1 // indirect
	github.com/godbus/dbus/v5 v5.1.0 // indirect
	github.com/gofiber/schema v1.2.0 // indirect
	github.com/gofiber/utils/v2 v2.0.0-beta.8 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang-jwt/jwt/v4 v4.5.2 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/google/go-querystring v1.1.0 // indirect
	github.com/goph/emperror v0.17.2 // indirect
	github.com/gorilla/websocket v1.5.3 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/hashicorp/go-version v1.7.0 // indirect
	github.com/hekmon/cunits/v2 v2.1.0 // indirect
	github.com/invopop/yaml v0.3.1 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/klauspost/compress v1.18.0 // indirect
	github.com/klauspost/cpuid/v2 v2.2.10 // indirect
	github.com/leodido/go-urn v1.4.0 // indirect
	github.com/lithammer/shortuuid/v3 v3.0.7 // indirect
	github.com/lufia/plan9stats v0.0.0-20211012122336-39d0f177ccd0 // indirect
	github.com/mailru/easyjson v0.9.0 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/meguminnnnnnnnn/go-openai v0.0.0-20250408071642-761325becfd6 // indirect
	github.com/minio/crc64nvme v1.0.1 // indirect
	github.com/minio/md5-simd v1.1.2 // indirect
	github.com/mitchellh/colorstring v0.0.0-20190213212951-d06e56a500db // indirect
	github.com/mmcdole/goxpp v1.1.1-0.20240225020742-a0c311522b23 // indirect
	github.com/moby/docker-image-spec v1.3.1 // indirect
	github.com/moby/sys/atomicwriter v0.1.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/mohae/deepcopy v0.0.0-20170929034955-c48cc78d4826 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/nikolalohinski/gonja v1.5.3 // indirect
	github.com/nu7hatch/gouuid v0.0.0-20131221200532-179d4d0c4d8d // indirect
	github.com/oklog/ulid v1.3.1 // indirect
	github.com/oklog/ulid/v2 v2.1.1 // indirect
	github.com/ollama/ollama v0.5.12 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.1.0 // indirect
	github.com/pelletier/go-toml/v2 v2.2.3 // indirect
	github.com/perimeterx/marshmallow v1.1.5 // indirect
	github.com/philhofer/fwd v1.1.3-0.20240916144458-20a13a1f6b7c // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/power-devops/perfstat v0.0.0-20210106213030-5aafc221ea8c // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/robfig/cron/v3 v3.0.1 // indirect
	github.com/rs/xid v1.6.0 // indirect
	github.com/sagikazarmark/locafero v0.7.0 // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	github.com/slongfield/pyfmt v0.0.0-20220222012616-ea85ff4c361f // indirect
	github.com/sony/gobreaker v1.0.0 // indirect
	github.com/sourcegraph/conc v0.3.0 // indirect
	github.com/spf13/afero v1.12.0 // indirect
	github.com/spf13/cast v1.7.1 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	github.com/swaggo/files/v2 v2.0.2 // indirect
	github.com/tadvi/systray v0.0.0-20190226123456-11a2b8fa57af // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.0 // indirect
	github.com/tinylib/msgp v1.3.0 // indirect
	github.com/tklauser/go-sysconf v0.3.12 // indirect
	github.com/tklauser/numcpus v0.6.1 // indirect
	github.com/twitchyliquid64/golang-asm v0.15.1 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/valyala/fastrand v1.1.0 // indirect
	github.com/valyala/histogram v1.2.0 // indirect
	github.com/vmihailenco/msgpack v4.0.4+incompatible // indirect
	github.com/x448/float16 v0.8.4 // indirect
	github.com/yargevad/filepathx v1.0.0 // indirect
	github.com/yosida95/uritemplate/v3 v3.0.2 // indirect
	github.com/yusufpapurcu/wmi v1.2.4 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.54.0 // indirect
	go.opentelemetry.io/otel v1.29.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp v1.22.0 // indirect
	go.opentelemetry.io/otel/metric v1.29.0 // indirect
	go.opentelemetry.io/otel/trace v1.29.0 // indirect
	go.uber.org/atomic v1.9.0 // indirect
	go.uber.org/dig v1.19.0 // indirect
	go.uber.org/multierr v1.10.0 // indirect
	go.uber.org/zap v1.26.0 // indirect
	golang.org/x/arch v0.14.0 // indirect
	golang.org/x/mod v0.23.0 // indirect
	golang.org/x/net v0.40.0 // indirect
	golang.org/x/sync v0.14.0 // indirect
	golang.org/x/term v0.32.0 // indirect
	golang.org/x/text v0.25.0 // indirect
	golang.org/x/tools v0.30.0 // indirect
	google.golang.org/appengine v1.6.8 // indirect
	google.golang.org/protobuf v1.36.6 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	gorm.io/datatypes v1.2.4 // indirect
	gorm.io/hints v1.1.0 // indirect
)
