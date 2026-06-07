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
	Project = "c6eme-42"
	zone    = "c6e"
)

func (s *server) updateZone(ctx context.Context, domains []string, ip, ipv6 string, clear bool) (err error, changed bool) {
	resp, err := s.dnsService.ResourceRecordSets.List(Project, zone).Context(ctx).Do()
	if err != nil {
		return
	}

	// Index existing A/AAAA records by name and type for quick lookup.
	existing := map[string]map[string]*dns.ResourceRecordSet{}
	for _, rrset := range resp.Rrsets {
		if rrset.Type != "A" && rrset.Type != "AAAA" {
			continue
		}
		if existing[rrset.Name] == nil {
			existing[rrset.Name] = map[string]*dns.ResourceRecordSet{}
		}
		existing[rrset.Name][rrset.Type] = rrset
	}

	additions := []*dns.ResourceRecordSet{}
	deletions := []*dns.ResourceRecordSet{}

	// reconcile records the minimal change to make name/typ resolve to value.
	// It only marks changed (and emits additions/deletions) when the record is
	// actually out of date, so an unchanged update is a no-op (-> "nochg").
	reconcile := func(name, typ, value string) {
		cur := existing[name][typ]
		if clear || value == "" {
			// Only remove the record on an explicit clear request.
			if clear && cur != nil {
				deletions = append(deletions, cur)
				changed = true
			}
			return
		}
		if cur != nil && len(cur.Rrdatas) == 1 && cur.Rrdatas[0] == value && cur.Ttl == 300 {
			return // already correct
		}
		if cur != nil {
			deletions = append(deletions, cur)
		}
		additions = append(additions, &dns.ResourceRecordSet{
			Type:    typ,
			Name:    name,
			Rrdatas: []string{value},
			Ttl:     300,
		})
		changed = true
	}

	for _, domain := range domains {
		reconcile(domain, "A", ip)
		reconcile(domain, "AAAA", ipv6)
	}

	if !changed {
		// Nothing to do; Cloud DNS rejects an empty change set.
		return nil, false
	}

	change := &dns.Change{
		Additions: additions,
		Deletions: deletions,
	}

	_, err = s.dnsService.Changes.Create(Project, zone, change).Context(ctx).Do()
	return err, changed
}
