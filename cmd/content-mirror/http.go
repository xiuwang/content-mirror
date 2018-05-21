package main

import (
	"fmt"
	htmltemplate "html/template"
	"log"
	"mime"
	"net/http"
	"strings"
	"text/template"

	"github.com/openshift/content-mirror/pkg/config"
)

const templateHTMLIndex = `
<!DOCTYPE html>
  <html lang='en'>
    <head>
      <meta charset='utf-8'>
      <title>Content Mirror</title>
    </head>
    <body>
      <h1>Available content</h1>
    <ul>
      {{- range .Upstreams }}
      {{- if .Repo }}
      <li><a href="/{{ .Name }}">{{ .Name }}</a> (<a href="/{{ .Name }}.repo">RPM repo</a>)
      {{- else }}
      <li><a href="/{{ .Name }}">{{ .Name }}</a>
      {{- end }}
      {{- end }}
    </ul>
  </body>
</html>
`

const templateUpstreamRepository = `
[{{ .Name }}]
id = {{ .Name }}
name = {{ .Name }}
baseurl = {{ .URL }}
enabled = 1
gpgcheck = 0
`

// ConfigAccessor returns the last valid configuration.
type ConfigAccessor interface {
	LastConfig() *config.CacheConfig
}

// NewHandlers returns the HTTP handlers for the provided config.
func NewHandlers(config ConfigAccessor) (http.Handler, error) {
	indexTemplate, err := htmltemplate.New("index").Parse(templateHTMLIndex)
	if err != nil {
		return nil, err
	}
	upstreamRepo, err := template.New("upstream-repo").Parse(templateUpstreamRepository)
	if err != nil {
		return nil, err
	}

	mux := http.NewServeMux()
	mux.Handle("/healthz", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		fmt.Fprintln(w, "ok")
	}))
	mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		lastConfig := config.LastConfig()
		if strings.Count(req.URL.Path, "/") == 1 && strings.HasSuffix(req.URL.Path, ".repo") {
			name := strings.TrimSuffix(req.URL.Path[1:], ".repo")
			for _, upstream := range lastConfig.Upstreams {
				if upstream.Name != name {
					continue
				}
				// not a candidate for being an RPM repository
				if !upstream.Repo {
					break
				}

				// output an RPM repository file dynamically
				upstream.URL = urlForRepo(req, &upstream)

				if err := upstreamRepo.Execute(w, &upstream); err != nil {
					log.Printf("error: Unable to write repository template %v", err)
				}
				return
			}
		}
		if req.URL.Path != "/" {
			http.NotFound(w, req)
			return
		}

		match, _ := hasAccept(req.Header.Get("Accept"), "text/html", "text/plain")
		if match == "text/html" {
			if err := indexTemplate.Execute(w, lastConfig); err != nil {
				log.Printf("error: Unable to write index template %v", err)
			}
			return
		}
		for _, upstream := range lastConfig.Upstreams {
			if !upstream.Repo {
				continue
			}
			upstream.URL = urlForRepo(req, &upstream)
			if err := upstreamRepo.Execute(w, &upstream); err != nil {
				log.Printf("error: Unable to write index template %v", err)
				break
			}
		}
	}))
	return mux, nil
}

func urlForRepo(req *http.Request, upstream *config.Upstream) string {
	url := *req.URL
	switch proto := req.Header.Get("X-Forwarded-Proto"); proto {
	case "https", "http":
		url.Scheme = proto
	default:
		if req.TLS != nil {
			url.Scheme = "https"
		} else {
			url.Scheme = "http"
		}
	}
	url.Host = req.Host
	url.Path = fmt.Sprintf("/%s", upstream.Name)
	return url.String()
}

func hasAccept(accept string, mediaTypes ...string) (string, bool) {
	for _, s := range strings.Split(accept, ",") {
		mediaType, _, err := mime.ParseMediaType(s)
		if err != nil {
			continue
		}
		for _, t := range mediaTypes {
			if mediaType == t {
				return mediaType, true
			}
		}
	}
	return "", false
}
