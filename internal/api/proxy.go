package api

import (
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"
)

func (api *API) spotifyProxy() http.Handler {
	if api.cfg.SpotifyBaseURL == "" {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "spotify base url is not configured", http.StatusServiceUnavailable)
		})
	}

	target, err := url.Parse(api.cfg.SpotifyBaseURL)
	if err != nil {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "spotify base url is invalid", http.StatusServiceUnavailable)
		})
	}

	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.Transport = api.proxyTransport()
	origDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		origDirector(req)
		req.URL.Path = singleJoin(target.Path, strings.TrimPrefix(req.URL.Path, "/v1/spotify"))
		req.Host = target.Host
	}
	return proxy
}

func (api *API) trackfileProxy() http.Handler {
	if api.cfg.TrackfileUpstream == "" {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "trackfile upstream is not configured", http.StatusServiceUnavailable)
		})
	}

	target, err := url.Parse(api.cfg.TrackfileUpstream)
	if err != nil {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "trackfile upstream is invalid", http.StatusServiceUnavailable)
		})
	}

	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.Transport = api.proxyTransport()
	origDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		origDirector(req)
		req.URL.Path = singleJoin(target.Path, strings.TrimPrefix(req.URL.Path, "/v1/trackfiles"))
		req.Host = target.Host
	}
	return proxy
}

func singleJoin(a, b string) string {
	a = strings.TrimRight(a, "/")
	b = strings.TrimLeft(b, "/")
	if a == "" {
		return "/" + b
	}
	if b == "" {
		return a
	}
	return a + "/" + b
}

func (api *API) proxyTransport() *http.Transport {
	return &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   api.cfg.ProxyTimeout,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: api.cfg.ProxyTimeout,
		ExpectContinueTimeout: 1 * time.Second,
	}
}
