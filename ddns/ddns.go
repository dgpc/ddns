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
	"fmt"
	"net"
	"net/http"
	"strings"

	"ddns/third_party/netutil"

	"cloud.google.com/go/datastore"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"google.golang.org/api/dns/v1"
)

type server struct {
	dnsService *dns.Service
	dsCli      *datastore.Client
	Router     *mux.Router

	// Dependencies, overridable in tests. Default to the real
	// implementations (see authorized / updateZone) in NewServer.
	authorize func(ctx context.Context, domain, token string) (bool, error)
	update    func(ctx context.Context, domains []string, ip, ipv6 string, clear bool) (err error, changed bool)
}

func NewServer(dnsService *dns.Service, dsCli *datastore.Client) *server {
	srv := &server{
		dnsService: dnsService,
		dsCli:      dsCli,
	}
	srv.authorize = srv.authorized
	srv.update = srv.updateZone

	r := mux.NewRouter()
	r.HandleFunc("/update", srv.UpdateHandler)
	r.HandleFunc("/nic/update", srv.UpdateHandler)
	r.HandleFunc("/dyn/dyndns.php", srv.UpdateHandler)
	r.HandleFunc("/update/{domain}/{token}", srv.UpdateHandler)
	r.HandleFunc("/update/{domain}/{token}/{ip}", srv.UpdateHandler)
	r.Use(handlers.ProxyHeaders)
	srv.Router = r

	return srv
}

func any(ss ...string) string {
	for _, s := range ss {
		if s != "" {
			return s
		}
	}
	return ""
}

// isDyndns2 reports whether the request arrived on one of the dyndns2 protocol
// endpoints. Clients such as ddclient expect those endpoints to answer with
// dyndns2 status codes ("good"/"nochg"/"badauth"/...) rather than this server's
// native OK/KO format, so they cannot understand a bare "OK" reply.
func isDyndns2(r *http.Request) bool {
	switch r.URL.Path {
	case "/nic/update", "/dyn/dyndns.php":
		return true
	}
	return false
}

// fail writes an error response in the dialect appropriate to the endpoint. On
// dyndns2 endpoints the result is conveyed by a status code in the body with an
// HTTP 200 (per the dyndns2 protocol); otherwise the native KO format is used,
// preserving the optional verbose detail line.
func fail(w http.ResponseWriter, dyndns2 bool, httpStatus int, dynCode, verboseMsg string, verbose bool) {
	if dyndns2 {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "%s\n", dynCode)
		return
	}
	w.WriteHeader(httpStatus)
	fmt.Fprint(w, "KO\n")
	if verbose {
		fmt.Fprint(w, verboseMsg)
	}
}

func (srv *server) UpdateHandler(w http.ResponseWriter, r *http.Request) {
	dyndns2 := isDyndns2(r)

	domain := any(r.FormValue("domains"), r.FormValue("hostname"))
	token := r.FormValue("token")
	if _, pass, ok := r.BasicAuth(); ok && token == "" {
		token = pass
	}
	ip := any(r.FormValue("ip"), r.FormValue("myip"))
	ipv6 := r.FormValue("ipv6")
	verbose := r.FormValue("verbose") == "true"
	clear := r.FormValue("clear") == "true"
	if clear {
		ip = ""
		ipv6 = ""
	}

	if domain == "" {
		v := mux.Vars(r)
		domain = v["domain"]
		token = v["token"]
		ip = v["ip"]
	}
	domains := strings.Split(domain, ",")

	w.Header().Set("Content-Type", "text/plain")

	if len(domains) == 0 {
		fail(w, dyndns2, http.StatusBadRequest, "notfqdn", "no domain(s) provided\n", verbose)
		return
	}

	for i, domain := range domains {
		domain = netutil.AbsDomainName([]byte(domain))
		domains[i] = domain
		if !netutil.IsDomainName(domain) {
			fail(w, dyndns2, http.StatusBadRequest, "notfqdn",
				fmt.Sprintf("invalid domain: %q\n", domain), verbose)
			return
		}

		auth, err := srv.authorize(r.Context(), domain, token)
		if err != nil {
			fail(w, dyndns2, http.StatusInternalServerError, "911",
				fmt.Sprintf("unable to verify token: %v\n", err), verbose)
			return
		}
		if !auth {
			fail(w, dyndns2, http.StatusUnauthorized, "badauth",
				fmt.Sprintf("incorrect token for domain %q\n", domain), verbose)
			return
		}
	}

	if ip == "" && ipv6 == "" {
		// Try to auto-detect client IP
		ip = r.RemoteAddr
		if ip == "" {
			fail(w, dyndns2, http.StatusInternalServerError, "911",
				"unable to determine client IP address\n", verbose)
			return
		}
		if host, _, err := net.SplitHostPort(ip); err == nil {
			ip = host // strip port if present
		}
	}

	if ipv6 != "" {
		// Validate explicit IPv6 address
		parsedIP := net.ParseIP(ipv6)
		if parsedIP == nil {
			fail(w, dyndns2, http.StatusBadRequest, "dnserr",
				fmt.Sprintf("invalid IP v6 address: %q\n", ipv6), verbose)
			return
		}
	}

	if ipv6 == "" {
		// No explicit IPv6 addr, determine whether IP is v4 or v6.
		parsedIP := net.ParseIP(ip)
		if parsedIP == nil {
			fail(w, dyndns2, http.StatusBadRequest, "dnserr",
				fmt.Sprintf("invalid IP address: %q\n", ip), verbose)
			return
		}
		if parsedIP.To4() == nil {
			ipv6 = ip
			ip = ""
		}
	}

	err, changed := srv.update(r.Context(), domains, ip, ipv6, clear)
	if err != nil {
		fail(w, dyndns2, http.StatusInternalServerError, "911",
			fmt.Sprintf("failed to update zone file: %v\n", err), verbose)
		return
	}

	w.WriteHeader(http.StatusOK)
	if dyndns2 {
		// dyndns2 reports one status line per host: "good <ip>" when the record
		// changed, "nochg <ip>" otherwise. Report whichever address was set.
		status := "good"
		if !changed {
			status = "nochg"
		}
		addr := any(ip, ipv6)
		for range domains {
			fmt.Fprintf(w, "%s %s\n", status, addr)
		}
		return
	}

	fmt.Fprint(w, "OK\n")
	if verbose {
		fmt.Fprintf(w, "%s\n", ip)
		fmt.Fprintf(w, "%s\n", ipv6)
		if changed {
			fmt.Fprintf(w, "UPDATED\n")
		} else {
			fmt.Fprintf(w, "NOCHANGE\n")
		}
	}
}
