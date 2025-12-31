package setup

import (
	"net/http"
)

// openRouterTransport adds required headers for OpenRouter API
type openRouterTransport struct {
	base http.RoundTripper
}

func (t *openRouterTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("HTTP-Referer", "https://github.com/robottwo/bishop")
	req.Header.Set("X-Title", "bishop - The Generative Shell")
	return t.base.RoundTrip(req)
}

// newOpenRouterClient creates an HTTP client with OpenRouter headers
func newOpenRouterClient() *http.Client {
	return &http.Client{
		Transport: &openRouterTransport{
			base: http.DefaultTransport,
		},
	}
}
