package backends

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	dns "google.golang.org/api/dns/v2beta1"
	"google.golang.org/api/option"
)

type cloudDNS struct {
	zone    string
	project string
	domain  string
	service *dns.Service
}

// NewCloudDNSZone initializes a new client for the specified CloudDNS zone.
func NewCloudDNSZone(
	ctx context.Context, name string, credentials []byte,
) (DNSZone, error) {
	// 1) Get client
	options := []option.ClientOption{
		option.WithCredentialsJSON(credentials),
		option.WithScopes(dns.NdevClouddnsReadwriteScope),
	}
	service, err := dns.NewService(ctx, options...)
	if err != nil {
		return nil, fmt.Errorf("Error connecting to Google CloudDNS: %s", err)
	}

	// 2) Parse credentials
	var fields map[string]string
	err = json.Unmarshal(credentials, &fields)
	if err != nil {
		return nil, fmt.Errorf("Error parsing service account credentials: %s", err)
	}

	// 3) Get project
	project, ok := fields["project_id"]
	if !ok {
		return nil, fmt.Errorf("Invalid format of service account credentials")
	}

	// 4) Get domain of zone
	zone, err := service.ManagedZones.Get(project, name).Do()
	if err != nil {
		return nil, fmt.Errorf("Unable to retrieve domain managed by zone '%s': %s", name, err)
	}

	return &cloudDNS{
		zone:    name,
		project: project,
		domain:  zone.DnsName,
		service: service,
	}, nil
}

func (client *cloudDNS) Domain() string {
	return strings.TrimRight(client.domain, ".")
}

func (client *cloudDNS) Equal(other DNSZone) bool {
	o, ok := other.(*cloudDNS)
	if !ok {
		return false
	}
	return client.zone == o.zone && client.project == o.project && client.domain == o.domain
}

func (client *cloudDNS) Update(record DNSRecord) error {
	return client.process(record,
		func(existing *dns.ResourceRecordSet, resource *dns.ResourceRecordSet) error {

			if existing.Ttl == resource.Ttl && len(existing.Rrdatas) == 1 &&
				existing.Rrdatas[0] == resource.Rrdatas[0] {
				// 1) Item already exists and does not need to be updated
				return nil
			}

			// 2) Item already exists and needs to be updated - we delete the existing record and
			// add the new one.
			change := &dns.Change{
				Additions: []*dns.ResourceRecordSet{resource},
				Deletions: []*dns.ResourceRecordSet{existing},
			}
			req := client.service.Changes.Create(client.project, client.zone, change)
			if _, err := req.Do(); err != nil {
				// 2.1) Unexpected error during update
				return err
			}

			// 3) Else, we're done
			return nil
		}, func(resource *dns.ResourceRecordSet) error {

			// 1) Just add the new resource
			change := &dns.Change{
				Additions: []*dns.ResourceRecordSet{resource},
			}
			req := client.service.Changes.Create(client.project, client.zone, change)
			_, err := req.Do()
			return err
		},
	)
}

func (client *cloudDNS) Delete(record DNSRecord) error {
	return client.process(record,
		func(existing *dns.ResourceRecordSet, resource *dns.ResourceRecordSet) error {

			// We just issue a change deleting the existing record
			change := &dns.Change{
				Deletions: []*dns.ResourceRecordSet{existing},
			}
			req := client.service.Changes.Create(client.project, client.zone, change)
			_, err := req.Do()
			return err

		}, func(resource *dns.ResourceRecordSet) error {

			// In this case, we don't have to do anything as our goal has already been fulfilled
			return nil
		},
	)
}

func (client *cloudDNS) process(
	record DNSRecord,
	processExisting func(*dns.ResourceRecordSet, *dns.ResourceRecordSet) error,
	processNonExisting func(*dns.ResourceRecordSet) error,
) error {
	dnsRecord := record.cloudDNSResource()

	// 1) Request existing records from CloudDNS
	request := client.service.ResourceRecordSets.List(client.project, client.zone)
	records, err := request.Name(dnsRecord.Name).Type(dnsRecord.Type).Do()
	if err != nil {
		// 2.1) Unexpected error during request
		return err
	}

	// 2) Check if this record already exists, and, if so, process existing
	if len(records.Rrsets) > 0 {
		item := records.Rrsets[0]
		if err := processExisting(item, dnsRecord); err != nil {
			return err
		}
		return nil
	}

	// 3) Item does not yet exist, process non existing
	if err := processNonExisting(dnsRecord); err != nil {
		return err
	}

	return nil
}

func (r DNSRecord) cloudDNSResource() *dns.ResourceRecordSet {
	// 1) Ensure correct domain format
	name := r.Name
	if !strings.HasSuffix(name, ".") {
		name += "."
	}

	// 2) Ensure correct rrdata for CNAME
	data := r.Data
	if r.Type == "CNAME" && !strings.HasSuffix(data, ".") {
		data += "."
	}

	return &dns.ResourceRecordSet{
		Name:    name,
		Type:    r.Type,
		Ttl:     int64(r.TTL),
		Rrdatas: []string{data},
	}
}
