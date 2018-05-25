package config

type CacheConfig struct {
	LocalPort        int
	CacheDir         string
	MaxCacheSize     string
	InactiveDuration string

	LogLevel string

	Frontends []Frontend
	Upstreams []Upstream
}

type Frontend struct {
	Listen          string
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
