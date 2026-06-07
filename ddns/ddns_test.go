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
	"net/http/httptest"
	"testing"
)

// testServer returns a server whose datastore/DNS dependencies are replaced
// with in-memory fakes, so handler behaviour can be tested without GCP. The
// fake accepts the token "hunter2" and reports every update as a change.
func testServer(changed bool) *server {
	srv := NewServer(nil, nil)
	srv.authorize = func(ctx context.Context, domain, token string) (bool, error) {
		return token == "hunter2", nil
	}
	srv.update = func(ctx context.Context, domains []string, ip, ipv6 string, clear bool) (error, bool) {
		return nil, changed
	}
	return srv
}

func TestUpdateValidation(t *testing.T) {
	srv := testServer(true)
	for name, tc := range map[string]struct {
		url    string
		status int
	}{
		"valid":                    {"https://ddns.c6e.me/update/c6e.me/hunter2/127.0.0.1", 200},
		"valid-params":             {"https://ddns.c6e.me/update?domains=c6e.me&token=hunter2&ip=127.0.0.1", 200},
		"valid-auto-ip":            {"https://ddns.c6e.me/update/c6e.me/hunter2", 200},
		"valid-auto-ip-params":     {"https://ddns.c6e.me/update?domains=c6e.me&token=hunter2", 200},
		"valid-multiple-domains":   {"https://ddns.c6e.me/update?domains=c6e.me,daave.com&token=hunter2", 200},
		"valid-ipv6":               {"https://ddns.c6e.me/update?domains=c6e.me&token=hunter2&ip=::1", 200},
		"valid-ipv4-and-v6":        {"https://ddns.c6e.me/update?domains=c6e.me&token=hunter2&ip=127.0.0.1&ipv6=::1", 200},
		"valid-ipv6-param":         {"https://ddns.c6e.me/update?domains=c6e.me&token=hunter2&ipv6=::1", 200},
		"invalid-ipv4":             {"https://ddns.c6e.me/update?domains=c6e.me&token=hunter2&ip=10.0.0.0.1", 400},
		"invalid-ipv4-blatant":     {"https://ddns.c6e.me/update?domains=c6e.me&token=hunter2&ip=text", 400},
		"invalid-ipv6":             {"https://ddns.c6e.me/update?domains=c6e.me&token=hunter2&ipv6=:::1", 400},
		"invalid-ipv6-blatant":     {"https://ddns.c6e.me/update?domains=c6e.me&token=hunter2&ipv6=text", 400},
		"invalid-domain":           {"https://ddns.c6e.me/update/c$e.me/hunter2", 400},
		"invalid-multiple-domains": {"https://ddns.c6e.me/update?domains=c6e.me,d@@ve.com&token=hunter2", 400},
		"domains-param-typo":       {"https://ddns.c6e.me/update?domain=c6e.me&token=hunter2", 400},
		"missing-auth":             {"https://ddns.c6e.me/update?domains=c6e.me", 401},
		"incorrect-auth":           {"https://ddns.c6e.me/update?domains=c6e.me&token=hunter7", 401},
	} {
		tc := tc
		t.Run(name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tc.url, nil)
			w := httptest.NewRecorder()
			srv.Router.ServeHTTP(w, req)
			status := w.Result().StatusCode
			if status != tc.status {
				t.Errorf("unexpected status code, got %d, want %d", status, tc.status)
			}
		})

	}
}

// TestDyndns2Responses verifies the dyndns2 endpoints answer with the protocol
// status codes that clients like ddclient understand, rather than OK/KO.
func TestDyndns2Responses(t *testing.T) {
	for name, tc := range map[string]struct {
		url      string
		bascPass string // if set, sent as HTTP Basic auth password (how ddclient authenticates)
		changed  bool
		want     string
	}{
		"good":          {"https://ddns.c6e.me/nic/update?hostname=c6e.me&token=hunter2&myip=127.0.0.1", "", true, "good 127.0.0.1\n"},
		"nochg":         {"https://ddns.c6e.me/nic/update?hostname=c6e.me&token=hunter2&myip=127.0.0.1", "", false, "nochg 127.0.0.1\n"},
		"good-ipv6":     {"https://ddns.c6e.me/nic/update?hostname=c6e.me&token=hunter2&myip=::1", "", true, "good ::1\n"},
		"basic-auth":    {"https://ddns.c6e.me/nic/update?hostname=c6e.me&myip=127.0.0.1", "hunter2", true, "good 127.0.0.1\n"},
		"badauth":       {"https://ddns.c6e.me/nic/update?hostname=c6e.me&token=wrong&myip=127.0.0.1", "", true, "badauth\n"},
		"notfqdn":       {"https://ddns.c6e.me/nic/update?hostname=c$e.me&token=hunter2&myip=127.0.0.1", "", true, "notfqdn\n"},
		"dyndns-php":    {"https://ddns.c6e.me/dyn/dyndns.php?hostname=c6e.me&token=hunter2&myip=127.0.0.1", "", true, "good 127.0.0.1\n"},
		"multi-domains": {"https://ddns.c6e.me/nic/update?hostname=c6e.me,daave.com&token=hunter2&myip=127.0.0.1", "", true, "good 127.0.0.1\ngood 127.0.0.1\n"},
	} {
		tc := tc
		t.Run(name, func(t *testing.T) {
			srv := testServer(tc.changed)
			req := httptest.NewRequest("GET", tc.url, nil)
			if tc.bascPass != "" {
				req.SetBasicAuth("c6e", tc.bascPass)
			}
			w := httptest.NewRecorder()
			srv.Router.ServeHTTP(w, req)
			// dyndns2 always responds 200; the result is in the body.
			if got := w.Result().StatusCode; got != 200 {
				t.Errorf("unexpected status code, got %d, want 200", got)
			}
			if got := w.Body.String(); got != tc.want {
				t.Errorf("unexpected body, got %q, want %q", got, tc.want)
			}
		})
	}
}
