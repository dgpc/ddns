/*
    Copyright 2020 Google LLC

    Licensed under the Apache License, Version 2.0 (the "License");
    you may not use this file except in compliance with the License.
    You may obtain a copy of the License at

        https://www.apache.org/licenses/LICENSE-2.0

    Unless required by applicable law or agreed to in writing, software
    distributed under the License is distributed on an "AS IS" BASIS,
    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
    See the License for the specific language governing permissions and
    limitations under the License.
*/

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
