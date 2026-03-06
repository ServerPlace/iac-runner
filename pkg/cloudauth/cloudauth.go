package cloudauth

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"google.golang.org/api/idtoken"
	"google.golang.org/api/option"
)

type Client struct {
	HTTP       *http.Client
	ServiceURL string // base URL (audience)
}

func NewCloudRunClient(ctx context.Context, serviceURL string) (*Client, error) {
	serviceURL = strings.TrimRight(serviceURL, "/")
	if serviceURL == "" {
		return nil, fmt.Errorf("cloudauth: serviceURL cannot be empty")
	}
	if _, err := url.Parse(serviceURL); err != nil {
		return nil, fmt.Errorf("cloudauth: invalid serviceURL: %w", err)
	}

	var opts []option.ClientOption
	if credFile := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"); credFile != "" {
		opts = append(opts, option.WithAuthCredentialsFile(option.ServiceAccount, credFile))
	}

	httpClient, err := idtoken.NewClient(ctx, serviceURL, opts...)
	if err != nil {
		return nil, fmt.Errorf("cloudauth: failed to create idtoken client: %w", err)
	}

	return &Client{
		HTTP:       httpClient,
		ServiceURL: serviceURL,
	}, nil
}

// DoWithTimeout é só um helper pra garantir deadline sem depender de timeout global no client.
func (c *Client) DoWithTimeout(ctx context.Context, req *http.Request, d time.Duration) (*http.Response, error) {
	ctx, cancel := context.WithTimeout(ctx, d)
	defer cancel()
	return c.HTTP.Do(req.WithContext(ctx))
}
