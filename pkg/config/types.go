package config

type CacheConfig struct {
	LocalPort        int
	CacheDir         string
	MaxCacheSize     string
	InactiveDuration string

	Frontends []Frontend
	Upstreams []Upstream
}

type Frontend struct {
	Port            int
	CertificatePath string
	KeyPath         string
}

type Upstream struct {
	Name  string
	URL   string
	Hosts []string

	Repo bool

	TLS               bool
	CACertificatePath string
	CertificatePath   string
	KeyPath           string
}
