package ddns

import (
	"context"

	"google.golang.org/api/dns/v1"
)

const (
	project = "c6eme-42"
	zone    = "c6e"
)

func (s *server) updateZone(ctx context.Context, domains []string, ip, ipv6 string, clear bool) (err error, changed bool) {
	resp, err := s.dnsService.ResourceRecordSets.List(project, zone).Context(ctx).Do()
	if err != nil {
		return
	}

	additions := []*dns.ResourceRecordSet{}
	deletions := []*dns.ResourceRecordSet{}

	for _, domain := range domains {
		for _, rrset := range resp.Rrsets {
			if rrset.Name == domain && rrset.Type == "A" && (clear || ip != "") {
				deletions = append(deletions, rrset)
				changed = true
			}
			if rrset.Name == domain && rrset.Type == "AAAA" && (clear || ipv6 != "") {
				deletions = append(deletions, rrset)
				changed = true
			}
		}

		if !clear && ip != "" {
			additions = append(additions, &dns.ResourceRecordSet{
				Type:    "A",
				Name:    domain,
				Rrdatas: []string{ip},
				Ttl:     300,
			})
			changed = true
		}

		if !clear && ipv6 != "" {
			additions = append(additions, &dns.ResourceRecordSet{
				Type:    "AAAA",
				Name:    domain,
				Rrdatas: []string{ipv6},
				Ttl:     300,
			})
			changed = true
		}
	}

	change := &dns.Change{
		Additions: additions,
		Deletions: deletions,
	}

	_, err = s.dnsService.Changes.Create(project, zone, change).Context(ctx).Do()
	return
}
