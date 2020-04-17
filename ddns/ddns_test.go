package ddns

import (
	"net/http/httptest"
	"testing"
)

func TestUpdateValidation(t *testing.T) {
	srv := NewServer(nil)
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
