package server

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strings"

	"ddns/netutil"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

func Router() *mux.Router {
	r := mux.NewRouter()
	r.HandleFunc("/update", UpdateHandler)
	r.HandleFunc("/update/{domain}/{token}", UpdateHandler)
	r.HandleFunc("/update/{domain}/{token}/{ip}", UpdateHandler)
	r.Use(handlers.ProxyHeaders)
	return r
}

func UpdateHandler(w http.ResponseWriter, r *http.Request) {
	domain := r.FormValue("domains")
	token := r.FormValue("token")
	ip := r.FormValue("ip")
	ipv6 := r.FormValue("ipv6")
	verbose := r.FormValue("verbose") == "true"
	// clear := r.FormValue("clear") == "true"
	if domain == "" {
		log.Println("attempt legacy no-parameter request")
		v := mux.Vars(r)
		domain = v["domain"]
		token = v["token"]
		ip = v["ip"]
	}
	domains := strings.Split(domain, ",")

	w.Header().Set("Content-Type", "text/plain")

	if len(domains) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		io.WriteString(w, "KO\n")
		if verbose {
			io.WriteString(w, "no domain(s) provided\n")
		}
		return
	}

	for _, domain := range domains {
		if !netutil.IsDomainName(domain) {
			w.WriteHeader(http.StatusBadRequest)
			io.WriteString(w, "KO\n")
			if verbose {
				io.WriteString(w, fmt.Sprintf("invalid domain: %q\n", domain))
			}
			return
		}
	}

	if token != "hunter2" { // TODO: auth
		w.WriteHeader(http.StatusUnauthorized)
		io.WriteString(w, "KO\n")
		if verbose {
			io.WriteString(w, "token should be 'hunter2' 🙈\n")
		}
		return
	}

	if ip == "" && ipv6 == "" {
		// Try to auto-detect client IP
		ip = r.RemoteAddr
		if ip == "" {
			w.WriteHeader(http.StatusInternalServerError)
			io.WriteString(w, "KO\n")
			if verbose {
				io.WriteString(w, "unable to determine client IP address\n")
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
			w.WriteHeader(http.StatusBadRequest)
			io.WriteString(w, "KO\n")
			if verbose {
				io.WriteString(w, fmt.Sprintf("invalid IP v6 address: %q\n", ipv6))
			}
			return
		}

		// TODO: removed this check since it breaks valid 4-to-6 addresses in the 0:0:0:0:0:ffff:* range
		// if parsedIP.To4() != nil {
		// 	w.WriteHeader(http.StatusBadRequest)
		// 	io.WriteString(w, "KO\n")
		// 	if verbose {
		// 		io.WriteString(w, fmt.Sprintf("IPv4 address provided in IPv6 field: %q\n", ipv6))
		// 	}
		// 	return
		// }
	}

	if ipv6 == "" {
		// No explicit IPv6 addr, determine whether IP is v4 or v6.
		parsedIP := net.ParseIP(ip)
		if parsedIP == nil {
			w.WriteHeader(http.StatusBadRequest)
			io.WriteString(w, "KO\n")
			if verbose {
				io.WriteString(w, fmt.Sprintf("invalid IP address: %q\n", ip))
			}
			return
		}
		if parsedIP.To4() == nil {
			ipv6 = ip
			ip = ""
		}
	}

	w.WriteHeader(http.StatusOK)
	io.WriteString(w, "OK\n")
	if verbose {
		io.WriteString(w, ip+"\n")
		io.WriteString(w, ipv6+"\n")
		io.WriteString(w, "UPDATED\n") // or NOCHANGE
	}
}
