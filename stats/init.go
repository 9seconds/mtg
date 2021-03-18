package stats

const (
	DefaultMetricPrefix = "mtg"

	DefaultStatsdMetricPrefix = DefaultMetricPrefix + "."
	DefaultStatsdTagFormat    = "datadog"

	MetricActiveConnection   = "active_connections"
	MetricSessionDuration    = "session_duration"
	MetricConcurrencyLimited = "concurrency_limited"
	MetricIPBlocklisted      = "ip_blocklisted"

	TagIPType = "ip_type"

	TagIPTypeIPv4 = "ipv4"
	TagIPTypeIPv6 = "ipv6"
)
