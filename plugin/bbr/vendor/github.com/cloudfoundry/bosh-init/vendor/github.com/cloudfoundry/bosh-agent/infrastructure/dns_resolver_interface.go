package infrastructure

type DNSResolver interface {
	LookupHost(dnsServers []string, endpoint string) (string, error)
}
