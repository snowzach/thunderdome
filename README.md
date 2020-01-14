# Thunderdome API 

This is a custodial wallet API for Lightning Network

## Compiling
This is designed as a go module aware program and thus requires go 1.11 or better
You can clone it anywhere, just run `make` inside the cloned directory to build

## Requirements
This does require a postgres database to be setup and reachable. It will attempt to create and migrate the database upon starting.

## Configuration
The configuration can be specified in a number of ways. By default you can create a json file and call it with the -c option
you can also specify environment variables that align with the config file values.

Example:
```json
{
	"logger": {
        "level": "debug"
	}
}
```
Can be set via an environment variable:
```
LOGGER_LEVEL=debug
```

### Options:
| Setting                                | Description                                                       | Default                            |
|----------------------------------------|-------------------------------------------------------------------|------------------------------------|
| logger.level                           | The default logging level                                         | "info"                             |
| logger.encoding                        | Logging format (console or json)                                  | "console"                          |
| logger.color                           | Enable color in console mode                                      | true                               |
| logger.disable_caller                  | Hide the caller source file and line number                       | false                              |
| logger.disable_stacktrace              | Hide a stacktrace on debug logs                                   | true                               |
| ---                                    | ---                                                               | ---                                |
| pidfile                                | Write a pidfile (only if specified)                               | ""                                 |
| profiler.enabled                       | Enable the debug pprof interface                                  | "false"                            |
| profiler.host                          | The profiler host address to listen on                            | ""                                 |
| profiler.port                          | The profiler port to listen on                                    | "6060"                             |
| ---                                    | ---                                                               | ---                                |
| dogstatsd.enabled                      | Enabled dogstatsd stats/events                                    | false                              |
| dogstatsd.host                         | dogstatsd agent host                                              | dogstatsd                          |
| dogstatsd.port                         | dogstatsd agent port                                              | 8125                               |
| dogstatsd.namespace                    | Namespace for metrics (prepend metric names)                      | "thunderdome."                     |
| dogstatsd.tags                         | Add the following tags to events and metrics                      | []                                 |
| ---                                    | ---                                                               | ---                                |
| server.host                            | The host address to listen on (blank=all addresses)               | ""                                 |
| server.port                            | The port number to listen on                                      | 8080                               |
| server.tls                             | Enable https/tls                                                  | false                              |
| server.devcert                         | Generate a development cert                                       | false                              |
| server.certfile                        | The HTTPS/TLS server certificate                                  | "server.crt"                       |
| server.keyfile                         | The HTTPS/TLS server key file                                     | "server.key"                       |
| server.log_requests                    | Log API requests                                                  | true                               |
| server.log_requests_body               | Log API requests body                                             | false                              |
| server.log_disabled_http               | Don't log these http api endpoints                                | ["/version"]                       |
| server.log_disabled_grpc               | Don't log these grpc api endpoints                                | ["/versionrpc.VersionRPC/Version"] |
| server.log_disabled_grpc_stream        | Don't log these grpc stream endpoints                             | []                                 |
| server.profiler_enabled                | Enable the profiler                                               | false                              |
| server.profiler_path                   | Where should the profiler be available                            | "/debug"                           |
| ---                                    | ---                                                               | ---                                |
| storage.type                           | The database type (supports postgres)                             | "postgres"                         |
| storage.username                       | The database username                                             | "postgres"                         |
| storage.password                       | The database password                                             | "password"                         |
| storage.host                           | Thos hostname for the database                                    | "postgres"                         |
| storage.port                           | The port for the database                                         | 5432                               |
| storage.database                       | The database                                                      | "thunderdome"                      |
| storage.database_test                  | The database used during go testing                               | "thunderdome_test"                 |
| storage.sslmode                        | The postgres sslmode to use                                       | "disable"                          |
| storage.retries                        | How many times to try to reconnect to the database on start       | 5                                  |
| storage.sleep_between_retriews         | How long to sleep between retries                                 | "7s"                               |
| storage.max_connections                | How many pooled connections to have                               | 80                                 |
| storage.wipe_confirm                   | Wipe the database during start                                    | false                              |
| ---                                    | ---                                                               | ---                                |
| redis.host                             | The redis server host                                             | "redis"                            |
| redis.port                             | The redis server port                                             | "6379"                             |
| redis.master_name                      | The redis server master name for HA                               | ""                                 |
| redis.password                         | The redis server password                                         | ""                                 |
| redis.index                            | The redis index numer                                             | 0                                  |
| redis.prefixes                         | Prefix all keys with the following keys                           | ["tdome"]                          |
| ---                                    | ---                                                               | ---                                |
| lnd.host                               | Lightning Node Host                                               | "lnd"                              |
| lnd.port                               | Lightning Node Port                                               | "10009"                            |
| lnd.tls_insecure                       | Ignore any tls issues when connecting to lnd                      | false                              |
| lnd.tls_cert                           | Lightning Node Server TLS Cert                                    | "tls.cert"                         |
| lnd.tls_host                           | Hostname to use for insecure tls                                  | ""                                 |
| lnd.macaroon                           | Lightning Node Client Macaroon                                    | "admin.macaroon"                   |
| lnd.unlock_password                    | The password to unlock the lnd wallet                             | "testtest"                         |
| lnd.health_check_interval              | Check lnd health status on this interval                          | "30s"                              |
| ---                                    | ---                                                               | ---                                |
| blocc.host                             | The blocc server host                                             | "blocc"                            |
| blocc.port                             | The blocc server port                                             | 8080                               |
| blocc.tls                              | Use TLS when talking to server                                    | false                              |
| blocc.tls_insecure                     | When using TLS, allow cert mismatch                               | false                              |
| blocc.tls_cert                         | TLS Certificate                                                   | "tls.cert"                         |
| blocc.tls_host                         | Override the hostname of the TLS cert                             | "blocc"                            |
| ---                                    | ---                                                               | ---                                |
| tdome.disabled                         | Shuts down the entire system and returns 503                      | false                              |
| tdome.disable_auth                     | Should authentication be disabled                                 | false                              |
| tdome.lock_new_accounts                | Should a new account be locked when created                       | true                               |
| tdome.firebase_credentials_file        | Path to the firebase credentials.json file for admin auth         | ""                                 |
| tdome.firebase_admin_role              | The default role looked at for cn_auth                            | "cn_role"                          |
| tdome.agent_secret                     | A secret that can be used to create account and payment requests  | "" = disabled                      |
| ---                                    | ---                                                               | ---                                |
| tdome.value_limit                      | The max amount you can send or request                            | 1000000                            |
| tdome.processing_fee_rate              | The percentage fee charged for paying and invoice 0.1 = 0.1%      | 0.0                                |
| tdome.network_fee_limit                | The limit we will accept for a withdraw fee                       | 40000                              |
| tdome.default_request_expires          | How long a payment request is good for (seconds)                  | 172800                             |
| tdome.create_generated_expires         | How long a generic invoice expiration will be in seconds          | 2592000                            |
| tdome.create_request_limit             | How many unpaid invoices a user can have                          | 5                                  |
| ---                                    | ---                                                               | ---                                |
| tdome.default_withdraw_target_blocks   | The default number of target blocks for confirmation on withdraw  | 6                                  |
| tdome.withdraw_fee_rate                | The percentage fee charged for a withdraw 0.1 = 0.1%              | 1.0                                |
| tdome.withdraw_fee_estimate            | The fee used for estimating withdraw transactions                 | 2000                               |
| tdome.tdome.withdraw_min               | Minimum amount of satoshis for a withdraw                         | 40000                              |
| ---                                    | ---                                                               | ---                                |
| tdome.topup_instant_enabled            | Allow crediting a user account with no confirmations              | false                              |
| tdome.topup_instant_user_count_limit   | The number of pending topup requests to allow another instant one | 2                                  |
| tdome.topup_instant_user_value_limit   | The maximum pending instant topup request amounts                 | 1200000                            |
| tdome.topup_instant_system_value_limit | The system wide max instant pending topups                        | 1200000                            |
| tdome.topup_fee_free                   | Credit the transaction fee to the users account on topup          | false                              |
| tdome.topup_fee_free_limit             | The max amount a user can be credited for fee free topup (in sat) | 40000                              |
| tdome.topup_alert_large                | Generate an alert when a topup received larger than this value    | 1500000                            |

## Data Storage
Data is stored in a postgres database

## TLS/HTTPS
You can enable https by setting the config option server.tls = true and pointing it to your keyfile and certfile.
To create a self-signed cert: `openssl req -new -newkey rsa:2048 -days 3650 -nodes -x509 -keyout server.key -out server.crt`

## Sanity Checking
This query is for checking inputs vs outputs and ensuring the user balance matches the transactions that have taken place
```
select 
(select sum(value) as total from ledger where direction = 'in' AND status = 'completed')
-
(select sum(value)+sum(network_fee)+sum(processing_fee) as total from ledger where direction = 'out' AND (status = 'completed' OR status = 'pending'))
-
(select sum(balance) from account)
AS delta
```
