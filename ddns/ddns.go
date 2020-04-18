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
	"fmt"
	"net"
	"net/http"
	"strings"

	"ddns/netutil"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"google.golang.org/api/dns/v1"
)

type server struct {
	dnsService *dns.Service
	Router     *mux.Router
}

func NewServer(dnsService *dns.Service) *server {
	srv := &server{
		dnsService: dnsService,
	}

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

func (srv *server) UpdateHandler(w http.ResponseWriter, r *http.Request) {
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
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "KO\n")
		if verbose {
			fmt.Fprintf(w, "no domain(s) provided\n")
		}
		return
	}

	for i, domain := range domains {
		if !netutil.IsDomainName(domain) {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "KO\n")
			if verbose {
				fmt.Fprintf(w, "invalid domain: %q\n", domain)
			}
			return
		}
		domains[i] = netutil.AbsDomainName([]byte(domain))
	}

	if token != "hunter2" { // TODO: auth
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintf(w, "KO\n")
		if verbose {
			fmt.Fprintf(w, "token should be 'hunter2' 🙈\n")
		}
		return
	}

	if ip == "" && ipv6 == "" {
		// Try to auto-detect client IP
		ip = r.RemoteAddr
		if ip == "" {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "KO\n")
			if verbose {
				fmt.Fprintf(w, "unable to determine client IP address\n")
			}
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
			// consider http.Error()
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "KO\n")
			if verbose {
				fmt.Fprintf(w, "invalid IP v6 address: %q\n", ipv6)
			}
			return
		}
	}

	if ipv6 == "" {
		// No explicit IPv6 addr, determine whether IP is v4 or v6.
		parsedIP := net.ParseIP(ip)
		if parsedIP == nil {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "KO\n")
			if verbose {
				fmt.Fprintf(w, "invalid IP address: %q\n", ip)
			}
			return
		}
		if parsedIP.To4() == nil {
			ipv6 = ip
			ip = ""
		}
	}

	err, changed := srv.updateZone(r.Context(), domains, ip, ipv6, clear)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "KO\n")
		if verbose {
			fmt.Fprintf(w, "failed to update zone file: %v\n", err)
		}
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "OK\n")
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
