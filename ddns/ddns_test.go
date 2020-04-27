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
	"net/http/httptest"
	"testing"
)

func TestUpdateValidation(t *testing.T) {
	srv := NewServer(nil, nil)
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
