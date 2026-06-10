package web

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	readability "codeberg.org/readeck/go-readability/v2"
)

func isHTTPURL(s string) bool {
	u, err := url.ParseRequestURI(s)
	if err != nil {
		return false
	}
	return u.Scheme == "http" || u.Scheme == "https"
}

func validateFetchURL(raw string) (*url.URL, error) {
	u, err := url.ParseRequestURI(raw)
	if err != nil {
		return nil, fmt.Errorf("invalid url: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("only http and https urls are allowed")
	}
	if u.Host == "" {
		return nil, fmt.Errorf("url missing host")
	}

	host := u.Hostname()
	if strings.EqualFold(host, "localhost") && !allowPrivateFetch() {
		return nil, fmt.Errorf("localhost urls are not allowed")
	}

	ips, err := net.LookupIP(host)
	if err != nil {
		return u, nil
	}
	if !allowPrivateFetch() {
		for _, ip := range ips {
			if isPrivateIP(ip) {
				return nil, fmt.Errorf("private network urls are not allowed")
			}
		}
	}
	return u, nil
}

func allowPrivateFetch() bool {
	return os.Getenv("ENOUGH_WEB_ALLOW_PRIVATE") == "1"
}

func isPrivateIP(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return true
	}
	privateRanges := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"127.0.0.0/8",
		"169.254.0.0/16",
		"0.0.0.0/8",
	}
	for _, cidr := range privateRanges {
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}
		if network.Contains(ip) {
			return true
		}
	}
	return false
}

// FetchPage downloads a URL and extracts readable full-page text.
func FetchPage(ctx context.Context, rawURL string) (Hit, error) {
	if err := ctx.Err(); err != nil {
		return Hit{}, err
	}

	u, err := validateFetchURL(rawURL)
	if err != nil {
		return Hit{}, err
	}

	timeout := time.Duration(fetchTimeoutSec) * time.Second
	article, err := readability.FromURL(u.String(), timeout, func(r *http.Request) {
		r.Header.Set("User-Agent", userAgent())
	})
	if err != nil {
		return Hit{URL: u.String()}, err
	}

	var content strings.Builder
	if err := article.RenderText(&content); err != nil {
		return Hit{URL: u.String()}, err
	}

	text := strings.TrimSpace(content.String())
	if text == "" {
		return Hit{URL: u.String()}, fmt.Errorf("no readable content extracted")
	}

	return Hit{
		Title:   article.Title(),
		URL:     u.String(),
		Content: text,
	}, nil
}

func userAgent() string {
	return "Enough/1.0 (+https://github.com/enough/enough)"
}
