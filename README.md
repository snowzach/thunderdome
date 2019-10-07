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
| Setting                                | Description                                                         | Default            |
|----------------------------------------|---------------------------------------------------------------------|--------------------|
| logger.level                           | The default logging level                                           | "info"             |
| logger.encoding                        | Logging format (console or json)                                    | "console"          |
| logger.color                           | Enable color in console mode                                        | true               |
| logger.disable_caller                  | Hide the caller source file and line number                         | false              |
| logger.disable_stacktrace              | Hide a stacktrace on debug logs                                     | true               |
| ---                                    | ---                                                                 | ---                |
| server.host                            | The host address to listen on (blank=all addresses)                 | ""                 |
| server.port                            | The port number to listen on                                        | 8900               |
| server.tls                             | Enable https/tls                                                    | false              |
| server.devcert                         | Generate a development cert                                         | false              |
| server.certfile                        | The HTTPS/TLS server certificate                                    | "server.crt"       |
| server.keyfile                         | The HTTPS/TLS server key file                                       | "server.key"       |
| server.log_requests                    | Log API requests                                                    | true               |
| server.profiler_enabled                | Enable the profiler                                                 | false              |
| server.profiler_path                   | Where should the profiler be available                              | "/debug"           |
| ---                                    | ---                                                                 | ---                |
| server.rest.enums_as_ints              | gRPC Gateway enums as ints                                          | false              |
| server.rest.emit_defaults              | gRPC Gateway emit default values                                    | true               |
| server.rest.orig_names                 | gRPC Gateway use original names                                     | true               |
| ---                                    | ---                                                                 | ---                |
| storage.type                           | The database type (supports postgres)                               | "postgres"         |
| storage.username                       | The database username                                               | "postgres"         |
| storage.password                       | The database password                                               | "password"         |
| storage.host                           | Thos hostname for the database                                      | "postgres"         |
| storage.port                           | The port for the database                                           | 5432               |
| storage.database                       | The database                                                        | "gorestapi"        |
| storage.sslmode                        | The postgres sslmode to use                                         | "disable"          |
| storage.retries                        | How many times to try to reconnect to the database on start         | 5                  |
| storage.sleep_between_retriews         | How long to sleep between retries                                   | "7s"               |
| storage.max_connections                | How many pooled connections to have                                 | 80                 |
| storage.wipe_confirm                   | Wipe the database during start                                      | false              |
| ---                                    | ---                                                                 | ---                |
| pidfile                                | Write a pidfile (only if specified)                                 | ""                 |
| profiler.enabled                       | Enable the debug pprof interface                                    | "false"            |
| profiler.host                          | The profiler host address to listen on                              | ""                 |
| profiler.port                          | The profiler port to listen on                                      | "6060"             |
| ---                                    | ---                                                                 | ---                |
| dogstats.enabled                       | Enabled sending dogstatsd metrics/events                            | false              |
| dogstats.host                          | dogstatds collector host                                            | dogstatsd          |
| dogstats.port                          | dogstatds collector port                                            | 8125               |
| dogstats.namespace                     | Prepend all metrics with namespace                                  | "thunderdome."     |
| dogstats.tags                          | Include tags with metrics/events ("name:value name:value")          | ""                 |
| -------------------------------------- | ------------------------------------------------------------------- | ------------------ |
| lnd.host                               | Lightning Node Host                                                 | "lnd"              |
| lnd.port                               | Lightning Node Port                                                 | "10009"            |
| lnd.tls_insecure                       | Ignore any tls issues when connecting to lnd                        | false              |
| lnd.tls_cert                           | Lightning Node Server TLS Cert                                      | "tls.cert"         |
| lnd.tls_host                           | Hostname to use for insecure tls                                    | ""                 |
| lnd.macaroon                           | Lightning Node Client Macaroon                                      | "admin.macaroon"   |
| lnd.unlock_password                    | The password to unlock the lnd wallet                               | "testtest"         |
| ---                                    | ---                                                                 | ---                |
| tdome.disabled                         | Shuts down the entire system and returns 503                        | false              |
| tdome.default_withdraw_target_blocks   | The default number of target blocks for confirmation on withdraw    | 6                  |
| tdome.disable_auth                     | Should authentication be disabled                                   | false              |
| tdome.lock_new_accounts                | Should a new account be locked when created                         | true               |
| tdome.firebase_credentials_file        | Path to the firebase credentials.json file for admin auth           | ""                 |
| tdome.value_limit                      | The max amount you can send or request                              | 500000             |
| tdome.processing_fee_rate              | The percentage fee charged for paying and invoice 0.1 = 0.1%        | 0.0                |
| tdome.withdraw_fee_estimate            | The fee used for estimating withdraw transactions                   | 2000               |
| tdome.withdraw_fee_rate                | The percentage fee charged for a withdraw 0.1 = 0.1%                | 0.1                |
| tdome.tdome.withdraw_min               | Minimum amount of satoshis for a withdraw                           | 40000              |
| tdome.network_fee_limit                | The limit we will accept for a withdraw fee                         | 40000              |
| tdome.topup_instant_enabled            | Allow crediting a user account with no confirmations                | false              |
| tdome.topup_instant_user_count_limit   | The number of pending topup requests to allow another instant one   | 2                  |
| tdome.topup_instant_user_value_limit   | The maximum pending instant topup request amounts                   | 100000             |
| tdome.topup_instant_system_value_limit | The system wide max instant pending topups                          | 1000000            |
| tdome.topup_fee_free                   | Credit the transaction fee to the users account on topup            | false              |
| tdome.topup_fee_free_limit             | The max amount a user can be credited for fee free topup (in sat)   | 40000              |
| tdome.default_request_expires          | How long a payment request is good for (seconds)                    | 86400              |
| tdome.agent_secret                     | A secret that can be used to create account and payment requests    | "" = disabled      |
| tdome.create_generated_expires         | How long a generic invoice expiration will be in seconds            | 2592000            |
|                                        |                                                                     |                    |

## Data Storage
Data is stored in a postgres database

## TLS/HTTPS
You can enable https by setting the config option server.tls = true and pointing it to your keyfile and certfile.
To create a self-signed cert: `openssl req -new -newkey rsa:2048 -days 3650 -nodes -x509 -keyout server.key -out server.crt`


## TODO
/balance
/decodeInvoices
/createinvoice
/getUserInvoices
/payInvoice
