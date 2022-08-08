// Stats package has implementations of [events.Observer] for different
// monitoring systems.
//
// Observer is a consumer of events produced by mtg. Consumers, defined
// in this package, process these events and provide information used by
// different monitoring system or time series databases.
package stats

const (
	// DefaultMetricPrefix defines a base prefix for all metrics.
	DefaultMetricPrefix = "mtg"

	// DefaultStatsdMetricPrefix defines a base prefix for metrics
	// which are passed to statsd.
	DefaultStatsdMetricPrefix = DefaultMetricPrefix + "."

	// DefaultStatsdTagFormat defines a format of tags for statsd
	// observer.
	DefaultStatsdTagFormat = "datadog"

	// MetricClientConnections defines a metric which is responsible for a
	// number of currently active connections established by client.
	//
	//     Type: gauge
	//     Tags:
	//       ip_family | A type of ip (ipv4 or ipv6) of the client.
	MetricClientConnections = "client_connections"

	// MetricTelegramConnections defines a metric which is responsible for
	// a count of active connections to Telegram servers.
	//
	//     Type: gauge
	//     Tags:
	//       telegram_ip | IP address of the telegram server.
	//       dc          | Index of the datacenter to connect to.
	MetricTelegramConnections = "telegram_connections"

	// MetricDomainFrontingConnections defines a metric which is
	// responsible for a count of active connections to a fronting domain.
	// Fronting domain is that one that is encoded in a secret.
	//
	//     Type: gauge
	//     Tags:
	//       ip_family | A type of IP (ipv4 or ipv6) that was used.
	MetricDomainFrontingConnections = "domain_fronting_connections"

	// MetricTelegramTraffic defines a metric for traffic (in bytes) that
	// is sent to and from Telegram servers.
	//
	//     Type: counter
	//     Tags:
	//       telegram_ip | IP address of the telegram server.
	//       dc          | Index of the datacenter
	//       direction   | Direction of the traffc flow. Values are
	//                   | 'to_client' and 'from_client'
	MetricTelegramTraffic = "telegram_traffic"

	// MetricDomainFrontingTraffic defines a metric for traffic (in bytes)
	// that is sent to and from fronting domain.
	//
	//     Type: counter
	//     Tags:
	//       direction   | Direction of the traffc flow. Values are
	//                   | 'to_client' and 'from_client'
	MetricDomainFrontingTraffic = "domain_fronting_traffic"

	// MetricDomainFronting defines a metric for a number of domain
	// fronting routing events.
	//
	//     Type: counter
	MetricDomainFronting = "domain_fronting"

	// MetricConcurrencyLimited defines a metric for a count of events,
	// when the client was blocked due to the concurrency limit.
	//
	//     Type: counter
	MetricConcurrencyLimited = "concurrency_limited"

	// MetricIPBlocklisted defines a metric for a count of events, when
	// client was blocked because her IP address was found in blocklists.
	//
	//     Type: counter
	MetricIPBlocklisted = "ip_blocklisted"

	// MetricReplayAttacks defines a metric for a count of events, when
	// mtg has detected a replay attack. Just a reminder: mtg immediately
	// routes a connection to a fronting domain if such event is detected.
	//
	//     Type: counter
	MetricReplayAttacks = "replay_attacks"

	// MetricIPListSize defines a metric for the size of the the ip list.
	//
	//     Type: gauge
	//     Tags:
	//       ip_list | 'allowlist' or 'blocklist'
	MetricIPListSize = "iplist_size"

	// TagIPFamily defines a name of the 'ip_family' tag and all values.
	TagIPFamily = "ip_family"

	// TagIPFamilyIPv4 defines a value of 'ip_family' of IPv4.
	TagIPFamilyIPv4 = "ipv4"

	// TagIPFamilyIPv6 defines a value of 'ip_family' of IPv6.
	TagIPFamilyIPv6 = "ipv6"

	// TagTelegramIP defines a name of the 'telegram_ip' tag.
	TagTelegramIP = "telegram_ip"

	// TagDC defines a name of the 'dc' tag.
	TagDC = "dc"

	// TagDirection defines a name of the 'direction' tag.
	TagDirection = "direction"

	// TagDirectionToClient defines that traffic is sent from Telegram to a
	// client.
	TagDirectionToClient = "to_client"

	// TagDirectionFromClient defines that traffic is sent from a client to
	// Telegram.
	TagDirectionFromClient = "from_client"

	// TagIPList defines a name of the 'ip_list' and all values.
	TagIPList = "ip_list"

	// TagIPListAllow defines a value of 'ip_list' of allowlist.
	TagIPListAllow = "allowlist"

	// TagIPListBlock defines a value of 'ip_list' of blocklist.
	TagIPListBlock = "blocklist"
)
