package backends

// DNSRecord describes a single DNS resource managed by a zone.
type DNSRecord struct {
	Name string
	Type string
	TTL  int
	Data string
}

// DNSZone represents a DNS zone managed by a particular provider.
type DNSZone interface {
	// Update adds or updates the given record in the zone. Requires all of the record's fields to
	// be set.
	Update(record DNSRecord) error

	// Delete removes the given record from the zone. Requires only the `Name` and `Type` field of
	// the record to be set. Fails only on unexpected errors. This does e.g. not include errors
	// occuring due to the zone not existing in the backend. It can be argued that the delete
	// operation is already carried out in this case.
	Delete(record DNSRecord) error

	// Equal returns whether the zone refers to the same zone as another DNSZone instance. This is
	// useful for detecting changes in DNSZones when updates are triggered.
	Equal(other DNSZone) bool

	// Domain returns the domain that the zone is managing (without trailing dot).
	Domain() string
}

// func (record DNSRecord) recordItems(domain string) []dnsResource {
// 	if domain == "" {
// 		panic("Domain must be set")
// 	}

// 	records := make([]dnsResource, 0)
// 	var primaryDNS string
// 	for i, host := range record.Hosts {
// 		dns := DNSFromHostAndDomain(host, domain)
// 		records = append(records, dnsResource{"A", dns, record.IP, record.TTL})
// 		if i == 0 {
// 			primaryDNS = dns
// 		}
// 	}

// 	for _, cname := range record.Cnames {
// 		dns := DNSFromHostAndDomain(cname, domain)
// 		records = append(records, dnsResource{"CNAME", dns, primaryDNS, record.TTL})
// 	}

// 	return records
// }

// DNSFromHostAndDomain returns the full DNS name from the given host and domain.
// func DNSFromHostAndDomain(host, domain string) string {
// 	if host == "@" {
// 		return fmt.Sprintf("%s", domain)
// 	}
// 	return fmt.Sprintf("%s.%s", host, domain)
// }
