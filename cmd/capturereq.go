package main

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"os"
	"strings"
	"sync"

	"github.com/miekg/dns"
)

func printreq(w http.ResponseWriter, r *http.Request) {
	fmt.Println("_______________________________________________________")
	requestDump, err := httputil.DumpRequest(r, true)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(string(requestDump))
	if w != nil {
		fmt.Fprint(w, string(requestDump))
	}
}

type transport struct {
	http.RoundTripper
}

func (t *transport) RoundTrip(req *http.Request) (resp *http.Response, err error) {
	resp, err = t.RoundTripper.RoundTrip(req)
	if err != nil {
		return nil, err
	}
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	err = resp.Body.Close()
	if err != nil {
		return nil, err
	}
	body := ioutil.NopCloser(bytes.NewReader(b))
	resp.Body = body
	rd, err := httputil.DumpResponse(resp, true)
	if err != nil {
		return nil, err
	}
	fmt.Println(string(rd))
	//fmt.Println("_______________________________________________________")
	return resp, nil
}

func publiclookup(d string) string {
	c := new(dns.Client)
	m := new(dns.Msg)
	m.SetQuestion(d+".", dns.TypeA)
	m.RecursionDesired = true
	r, _, err := c.Exchange(m, os.Getenv("DNS_SERVER")+":53")
	if err != nil {
		fmt.Println(err)
		return ""
	}
	if r.Rcode != dns.RcodeSuccess {
		return ""
	}
	for _, a := range r.Answer {
		if _, ok := a.(*dns.A); ok {
			return a.(*dns.A).A.String()
		}
		if _, ok := a.(*dns.CNAME); ok {
			return a.(*dns.CNAME).Target
		}
	}
	return ""
}

func proxyforhost(h string) (string, error) {
	ips, err := Lookup(h)
	if err != nil {
		return "", err
	}
	if len(ips) == 0 {
		ip := publiclookup(h)
		if ip == "" {
			return h, fmt.Errorf("Host not found: %s", h)
		}
		return ip, nil
	}
	return ips[0], nil
}

func proxyreq(w http.ResponseWriter, r *http.Request) {
	phostp := strings.Split(r.Host, ":")
	printreq(nil, r)
	ph, err := proxyforhost(phostp[0])
	if err != nil {
		fmt.Println(err)
		return
	}
	if ph == "127.0.0.1" || ph == "localhost" {
		fmt.Println("Loop detected")
		return
	}
	d := func(req *http.Request) {
		if r.TLS != nil && r.TLS.HandshakeComplete {
			req.URL.Scheme = "https"
		} else {
			req.URL.Scheme = "http"
		}
		if len(phostp) > 1 {
			req.URL.Host = ph + ":" + phostp[1]
		} else {
			req.URL.Host = ph
		}
		fmt.Printf("\n\nProxying to: %s\n\n", req.URL.Host)
		req.Host = r.Host
		req.URL.Path = r.URL.Path
		//req.Proto = "HTTP/1.1"
		//req.Header.Set("Connection", "")
	}
	e := func(w http.ResponseWriter, r *http.Request, e error) {
		http.Error(w, e.Error(), http.StatusBadGateway)
	}
	defaultTransport := http.DefaultTransport.(*http.Transport)
	customTransport := &http.Transport{
		Proxy:                 defaultTransport.Proxy,
		DialContext:           defaultTransport.DialContext,
		MaxIdleConns:          defaultTransport.MaxIdleConns,
		IdleConnTimeout:       defaultTransport.IdleConnTimeout,
		ExpectContinueTimeout: defaultTransport.ExpectContinueTimeout,
		TLSHandshakeTimeout:   defaultTransport.TLSHandshakeTimeout,
		TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
	}
	p := &httputil.ReverseProxy{
		Director:     d,
		ErrorHandler: e,
		Transport:    &transport{customTransport},
	}
	p.ServeHTTP(w, r)
}

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		proxyreq(w, r)
	})
	httpIps := strings.Split(os.Getenv("HTTP_PORTS"), ",")
	httpsIps := strings.Split(os.Getenv("HTTPS_PORTS"), ",")
	wg := new(sync.WaitGroup)
	for _, p := range httpIps {
		wg.Add(1)
		go func(p string) {
			log.Printf("Starting HTTP server on %s\n", p)
			log.Fatal(http.ListenAndServe(":"+p, nil))
			wg.Done()
		}(p)
	}
	for _, p := range httpsIps {
		wg.Add(1)
		go func(p string) {
			log.Printf("Starting HTTPS server on %s\n", p)
			log.Fatal(http.ListenAndServeTLS(":"+p, os.Getenv("CERT_FILE"), os.Getenv("KEY_FILE"), nil))
			wg.Done()
		}(p)
	}
	wg.Wait()
}
