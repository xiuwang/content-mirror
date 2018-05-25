package config

import (
	"fmt"
	"log"
	"net"
	"net/url"
	"strings"

	"github.com/go-ini/ini"
)

type RPMRepositorySection struct {
	ID            string `ini:"id"`
	Name          string
	BaseURL       string `ini:"baseurl"`
	Enabled       int
	SSLVerify     bool   `ini:"sslverify"`
	SSLClientKey  string `ini:"sslclientkey"`
	SSLClientCert string `ini:"sslclientcert"`
}

func LoadRPMRepoUpstreams(iniFile string) ([]Upstream, error) {
	var upstreams []Upstream
	cfg, err := ini.Load(iniFile)
	if err != nil {
		return nil, err
	}
	for _, section := range cfg.Sections() {
		if !section.Haskey("baseurl") {
			continue
		}
		repo := &RPMRepositorySection{
			ID:      section.Name(),
			Enabled: 1,
		}
		if err := section.MapTo(repo); err != nil {
			return nil, fmt.Errorf("%s can't load section %s: %v", iniFile, section.Name(), err)
		}
		if repo.Enabled == 0 {
			continue
		}
		var urls []*url.URL
		for _, u := range strings.Split(repo.BaseURL, " ") {
			u = strings.TrimSpace(u)
			if len(u) == 0 {
				continue
			}
			url, err := url.Parse(u)
			if err != nil {
				return nil, fmt.Errorf("repo %s has a base URL that is not a valid URL: %v", iniFile, u)
			}
			if !strings.HasSuffix(url.Path, "/") {
				url.Path += "/"
			}
			urls = append(urls, url)
		}
		if len(urls) == 0 {
			return nil, fmt.Errorf("repo %s has no baseurls", iniFile)
		}
		var hosts []string
		proxyPassURL := urls[0]
		for _, url := range urls {
			if url.Path == proxyPassURL.Path {
				if url.Scheme == "https" {
					if _, _, err := net.SplitHostPort(url.Host); err != nil {
						hosts = append(hosts, net.JoinHostPort(url.Host, "443"))
					} else {
						hosts = append(hosts, url.Host)
					}
				} else {
					hosts = append(hosts, url.Host)
				}
			}
		}
		if len(hosts) != len(urls) {
			log.Printf("one or more baseurls were omitted because they don't have a consistent path: %s", proxyPassURL.Path)
		}
		proxyPassURL.Host = repo.ID

		upstream := Upstream{
			Repo:  true,
			Name:  repo.ID,
			Hosts: hosts,
			URL:   proxyPassURL.String(),
		}
		if len(repo.SSLClientCert) > 0 {
			upstream.TLS = true
			upstream.CertificatePath = makePathRelativeToFile(iniFile, repo.SSLClientCert)
			upstream.KeyPath = makePathRelativeToFile(iniFile, repo.SSLClientKey)
		}
		upstreams = append(upstreams, upstream)
	}
	return upstreams, nil
}
