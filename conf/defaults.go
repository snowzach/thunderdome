package conf

import (
	"strings"

	config "github.com/spf13/viper"
)

func init() {

	// Sets up the config file, environment etc
	config.SetTypeByDefaultValue(true)                      // If a default value is []string{"a"} an environment variable of "a b" will end up []string{"a","b"}
	config.AutomaticEnv()                                   // Automatically use environment variables where available
	config.SetEnvKeyReplacer(strings.NewReplacer(".", "_")) // Environement variables use underscores instead of periods

	// Logger Defaults
	config.SetDefault("logger.level", "info")
	config.SetDefault("logger.encoding", "console")
	config.SetDefault("logger.color", true)
	config.SetDefault("logger.dev_mode", true)
	config.SetDefault("logger.disable_caller", false)
	config.SetDefault("logger.disable_stacktrace", true)

	// Pidfile
	config.SetDefault("pidfile", "")

	// Profiler config
	config.SetDefault("profiler.enabled", false)
	config.SetDefault("profiler.host", "")
	config.SetDefault("profiler.port", "6060")

	// Datadog DogStatsD Configuration
	config.SetDefault("dogstatsd.enabled", false)
	config.SetDefault("dogstatsd.host", "dogstatsd")
	config.SetDefault("dogstatsd.port", 8125)
	config.SetDefault("dogstatsd.namespace", "thunderdome.")
	config.SetDefault("dogstatsd.tags", []string{})

	// Server Configuration
	config.SetDefault("server.host", "")
	config.SetDefault("server.port", "8080")
	config.SetDefault("server.tls", false)
	config.SetDefault("server.devcert", false)
	config.SetDefault("server.certfile", "server.crt")
	config.SetDefault("server.keyfile", "server.key")
	config.SetDefault("server.log_requests", true)
	config.SetDefault("server.log_requests_body", false)
	config.SetDefault("server.log_disabled_http", []string{"/version"})
	config.SetDefault("server.log_disabled_grpc", []string{"/versionrpc.VersionRPC/Version"})
	config.SetDefault("server.log_disabled_grpc_stream", []string{})
	config.SetDefault("server.profiler_enabled", false)
	config.SetDefault("server.profiler_path", "/debug")

	// Database Settings
	config.SetDefault("storage.type", "postgres")
	config.SetDefault("storage.username", "postgres")
	config.SetDefault("storage.password", "password")
	config.SetDefault("storage.host", "postgres")
	config.SetDefault("storage.port", 5432)
	config.SetDefault("storage.database", "thunderdome")
	config.SetDefault("storage.database_test", "thunderdome_test")
	config.SetDefault("storage.sslmode", "disable")
	config.SetDefault("storage.retries", 5)
	config.SetDefault("storage.sleep_between_retries", "7s")
	config.SetDefault("storage.max_connections", 80)
	config.SetDefault("storage.wipe_confirm", false)

	config.SetDefault("lnd.host", "lnd")
	config.SetDefault("lnd.port", 10009)
	config.SetDefault("lnd.tls_insecure", false)
	config.SetDefault("lnd.tls_cert", "tls.cert")
	config.SetDefault("lnd.tls_host", "")
	config.SetDefault("lnd.macaroon", "admin.macaroon")
	config.SetDefault("lnd.unlock_password", "testtest")

	config.SetDefault("blocc.host", "blocc")
	config.SetDefault("blocc.port", 8080)
	config.SetDefault("blocc.tls", false)
	config.SetDefault("blocc.tls_insecure", true)
	config.SetDefault("blocc.tls_cert", "tls.cert")
	config.SetDefault("blocc.tls_host", "blocc")

	config.SetDefault("tdome.disabled", false)
	config.SetDefault("tdome.min_withdraw", 40000)
	config.SetDefault("tdome.default_withdraw_target_blocks", 6)
	config.SetDefault("tdome.disable_auth", false)
	config.SetDefault("tdome.lock_new_accounts", false)
	config.SetDefault("tdome.firebase_credentials_file", "")
	config.SetDefault("tdome.value_limit", 500000)
	config.SetDefault("tdome.processing_fee_rate", 0.0)
	config.SetDefault("tdome.withdraw_fee_rate", 1.0)
	config.SetDefault("tdome.network_fee_limit", 40000)
	config.SetDefault("tdome.topup_instant", false)
	config.SetDefault("tdome.topup_fee_free", false)
	config.SetDefault("tdome.topup_fee_free_limit", 40000)
	config.SetDefault("tdome.default_request_expires", 86400)
	config.SetDefault("tdome.agent_secret", "") // If left blank, it cannot be used
	config.SetDefault("tdome.create_generated_expires", 2592000)

}
