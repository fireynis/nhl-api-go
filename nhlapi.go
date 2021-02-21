package nhlAPI

import (
	"context"
	"encoding/json"
	"fmt"
	"golang.org/x/time/rate"
	"io"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"strings"
	"time"
)

const nhlApiUrlV1 = "https://statsapi.web.nhl.com/api/v1/"

// API holds the configuration for the current API client. A client should not
// be modified concurrently. Struct copied from Cloudflare SDK and modified.
type API struct {
	BaseURL     string
	UserAgent   string
	headers     http.Header
	httpClient  *http.Client
	rateLimiter *rate.Limiter
	retryPolicy RetryPolicy
	logger      Logger
}

// RetryPolicy specifies number of retries and min/max retry delays
// This config is used when the client exponentially backs off after errored requests.
// Struct copied from Cloudflare.
type RetryPolicy struct {
	MaxRetries    int
	MinRetryDelay time.Duration
	MaxRetryDelay time.Duration
}

// Logger defines the interface this library needs to use logging
// This is a subset of the methods implemented in the log package
type Logger interface {
	Printf(format string, v ...interface{})
}

type Response struct {
	Copyright string `json:"copyright"`
}

type ErrorResponse struct {
	MessageNumber int    `json:"messageNumber"`
	Message       string `json:"message"`
}

// newClient provides shared logic for New and NewWithUserServiceKey
func newClient(opts ...Option) (*API, error) {
	silentLogger := log.New(ioutil.Discard, "", log.LstdFlags)

	api := &API{
		BaseURL:     nhlApiUrlV1,
		headers:     make(http.Header),
		rateLimiter: rate.NewLimiter(rate.Limit(4), 1), // 4rps equates to default api limit (1200 req/5 min)
		retryPolicy: RetryPolicy{
			MaxRetries:    3,
			MinRetryDelay: time.Duration(1) * time.Second,
			MaxRetryDelay: time.Duration(30) * time.Second,
		},
		logger: silentLogger,
	}

	err := api.parseOptions(opts...)
	if err != nil {
		return nil, fmt.Errorf("options parsing failed %w", err)
	}

	// Fall back to http.DefaultClient if the package user does not provide
	// their own.
	if api.httpClient == nil {
		api.httpClient = http.DefaultClient
	}

	return api, nil
}

// New creates a new Cloudflare v4 API client.
func New(opts ...Option) (*API, error) {
	api, err := newClient(opts...)
	if err != nil {
		return nil, err
	}

	return api, nil
}

func (api *API) makeRequest(ctx context.Context, method, uri string, headers http.Header) ([]byte, error) {
	var err error

	var resp *http.Response
	var respErr error
	var reqBody io.Reader
	var respBody []byte
	for i := 0; i <= api.retryPolicy.MaxRetries; i++ {
		if i > 0 {
			// expect the backoff introduced here on errored requests to dominate the effect of rate limiting
			// don't need a random component here as the rate limiter should do something similar
			// nb time duration could truncate an arbitrary float. Since our inputs are all int, we should be ok
			sleepDuration := time.Duration(math.Pow(2, float64(i-1)) * float64(api.retryPolicy.MinRetryDelay))

			if sleepDuration > api.retryPolicy.MaxRetryDelay {
				sleepDuration = api.retryPolicy.MaxRetryDelay
			}
			// useful to do some simple logging here, maybe introduce levels later
			api.logger.Printf("Sleeping %s before retry attempt number %d for request %s %s", sleepDuration.String(), i, method, uri)
			time.Sleep(sleepDuration)

		}
		err = api.rateLimiter.Wait(context.TODO())
		if err != nil {
			return nil, fmt.Errorf("error caused by request rate limiting %w", err)
		}
		resp, respErr = api.request(ctx, method, uri, reqBody, headers)

		// retry if the server is rate limiting us or if it failed
		// assumes server operations are rolled back on failure
		if respErr != nil || resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
			// if we got a valid http response, try to read body so we can reuse the connection
			// see https://golang.org/pkg/net/http/#Client.Do
			if respErr == nil {
				respBody, err = ioutil.ReadAll(resp.Body)
				_ = resp.Body.Close()

				respErr = fmt.Errorf("could not read response body %w", err)

				api.logger.Printf("Request: %s %s got an error response %d: %s\n", method, uri, resp.StatusCode,
					strings.Replace(strings.Replace(string(respBody), "\n", "", -1), "\t", "", -1))
			} else {
				api.logger.Printf("Error performing request: %s %s : %s \n", method, uri, respErr.Error())
			}
			continue
		} else {
			respBody, err = ioutil.ReadAll(resp.Body)
			_ = resp.Body.Close()
			if err != nil {
				return nil, fmt.Errorf("could not read response body %w", err)
			}
			break
		}
	}
	if respErr != nil {
		return nil, respErr
	}

	switch {
	case resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusMultipleChoices:
	case resp.StatusCode == http.StatusUnauthorized:
		return nil, errorFromResponse(resp.StatusCode, respBody)
	case resp.StatusCode == http.StatusForbidden:
		return nil, errorFromResponse(resp.StatusCode, respBody)
	case resp.StatusCode == http.StatusServiceUnavailable,
		resp.StatusCode == http.StatusBadGateway,
		resp.StatusCode == http.StatusGatewayTimeout,
		resp.StatusCode == 522,
		resp.StatusCode == 523,
		resp.StatusCode == 524:
		return nil, fmt.Errorf("HTTP status %d: service failure", resp.StatusCode)
	// This isn't a great solution due to the way the `default` case is
	// a catch all and that the `filters/validate-expr` returns a HTTP 400
	// yet the clients need to use the HTTP body as a JSON string.
	case resp.StatusCode == 400 && strings.HasSuffix(resp.Request.URL.Path, "/filters/validate-expr"):
		return nil, fmt.Errorf("%s", respBody)
	default:
		var s string
		if respBody != nil {
			s = string(respBody)
		}
		return nil, fmt.Errorf("HTTP status %d: content %q", resp.StatusCode, s)
	}

	return respBody, nil
}

// request makes a HTTP request to the given API endpoint, returning the raw
// *http.Response, or an error if one occurred. The caller is responsible for
// closing the response body.
func (api *API) request(ctx context.Context, method, uri string, reqBody io.Reader, headers http.Header) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, api.BaseURL+uri, reqBody)
	if err != nil {
		return nil, fmt.Errorf("HTTP request creation failed %w", err)
	}

	combinedHeaders := make(http.Header)
	copyHeader(combinedHeaders, api.headers)
	copyHeader(combinedHeaders, headers)
	req.Header = combinedHeaders

	if api.UserAgent != "" {
		req.Header.Set("User-Agent", api.UserAgent)
	}

	if req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := api.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed %w", err)
	}

	return resp, nil
}

// copyHeader copies all headers for `source` and sets them on `target`.
// based on https://godoc.org/github.com/golang/gddo/httputil/header#Copy
func copyHeader(target, source http.Header) {
	for k, vs := range source {
		target[k] = vs
	}
}

// errorFromResponse returns a formatted error from the status code and error messages
// from the response body.
func errorFromResponse(statusCode int, respBody []byte) error {
	var r ErrorResponse
	err := json.Unmarshal(respBody, &r)
	if err != nil {
		return fmt.Errorf(errUnmarshalError+"%w", err)
	}

	return fmt.Errorf("HTTP status %d: %d - %s", statusCode, r.MessageNumber, r.Message)
}
