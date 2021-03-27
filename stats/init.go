package stats

const (
	DefaultMetricPrefix = "mtg"

	DefaultStatsdMetricPrefix = DefaultMetricPrefix + "."
	DefaultStatsdTagFormat    = "datadog"

	MetricClientConnections           = "client_connections"
	MetricTelegramConnections         = "telegram_connections"
	MetricDomainDisguisingConnections = "domain_disguising_connections"

	MetricTelegramTraffic         = "telegram_traffic"
	MetricDomainDisguisingTraffic = "domain_disguising_traffic"

	MetricDomainDisguising   = "domain_disguising"
	MetricConcurrencyLimited = "concurrency_limited"
	MetricIPBlocklisted      = "ip_blocklisted"
	MetricReplayAttacks      = "replay_attacks"

	TagIPFamily     = "ip_family"
	TagIPFamilyIPv4 = "ipv4"
	TagIPFamilyIPv6 = "ipv6"

	TagTelegramIP = "telegram_ip"

	TagDC = "dc"

	TagDirection           = "direction"
	TagDirectionToClient   = "to_client"
	TagDirectionFromClient = "from_client"
)
