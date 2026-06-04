module custody-flow-e2e

go 1.24

require (
	offline-client-wails v0.0.0
	offline-server v0.0.0
)

require github.com/gorilla/websocket v1.5.3

require (
	github.com/btcsuite/btcd/btcec/v2 v2.3.4 // indirect
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.0.1 // indirect
	github.com/ebfe/scard v0.0.0-20241214075232-7af069cabc25 // indirect
	github.com/ethereum/go-ethereum v1.15.9 // indirect
	github.com/fsnotify/fsnotify v1.9.0 // indirect
	github.com/go-viper/mapstructure/v2 v2.2.1 // indirect
	github.com/golang-jwt/jwt/v5 v5.2.2 // indirect
	github.com/holiman/uint256 v1.3.2 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/mattn/go-sqlite3 v1.14.22 // indirect
	github.com/pelletier/go-toml/v2 v2.2.3 // indirect
	github.com/sagikazarmark/locafero v0.7.0 // indirect
	github.com/sourcegraph/conc v0.3.0 // indirect
	github.com/spf13/afero v1.12.0 // indirect
	github.com/spf13/cast v1.7.1 // indirect
	github.com/spf13/pflag v1.0.6 // indirect
	github.com/spf13/viper v1.20.1 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	go.uber.org/multierr v1.10.0 // indirect
	go.uber.org/zap v1.27.0 // indirect
	golang.org/x/crypto v0.38.0 // indirect
	golang.org/x/sys v0.33.0 // indirect
	golang.org/x/text v0.25.0 // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.2.1 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	gorm.io/driver/sqlite v1.5.7 // indirect
	gorm.io/gorm v1.30.0 // indirect
)

replace offline-client-wails => ../../offline-client/offline-client-wails

replace offline-server => ../../offline-server
