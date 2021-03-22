package stats

const (
	DefaultMetricPrefix = "mtg"

	DefaultStatsdMetricPrefix = DefaultMetricPrefix + "."
	DefaultStatsdTagFormat    = "datadog"

	MetricClientConnections   = "client_connections"
	MetricTelegramConnections = "telegram_connections"
	MetricTraffic             = "traffic"
	MetricSessionDuration     = "session_duration"
	MetricSessionTraffic      = "session_traffic"
	MetricConcurrencyLimited  = "concurrency_limited"
	MetricIPBlocklisted       = "ip_blocklisted"

	TagIPType     = "ip_type"
	TagTelegramIP = "ip"
	TagDC         = "dc"
	TagDirection  = "direction"

	TagIPTypeIPv4        = "ipv4"
	TagIPTypeIPv6        = "ipv6"
	TagDirectionTelegram = "telegram"
	TagDirectionClient   = "client"
)
