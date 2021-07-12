package pagerduty

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"

	"github.com/PagerDuty/go-pagerduty"
)

const (
	apiEndpoint         = "https://api.pagerduty.com"
	v2EventsAPIEndpoint = "https://events.pagerduty.com"
)

// The type of authentication to use with the API client
type authType int

const (
	// Account/user API token authentication
	apiToken authType = iota

	// OAuth token authentication
	oauthToken
)

// ExtendedClient mimics github.com/PagerDuty/go-pagerduty/client.go
type ExtendedClient struct {
	*pagerduty.Client

	authToken           string
	apiEndpoint         string
	v2EventsAPIEndpoint string

	// Authentication type to use for API
	authType authType

	// HTTPClient is the HTTP client used for making requests against the
	// PagerDuty API. You can use either *http.Client here, or your own
	// implementation.
	HTTPClient pagerduty.HTTPClient
}

func NewExtendedClient(authToken string, options ...pagerduty.ClientOptions) *ExtendedClient {
	client := pagerduty.NewClient(authToken, options...)

	return &ExtendedClient{
		Client: client,

		apiEndpoint:         apiEndpoint,
		v2EventsAPIEndpoint: v2EventsAPIEndpoint,
		authType:            apiToken,
		authToken:           authToken,

		HTTPClient: client.HTTPClient,
	}
}

func (c *ExtendedClient) delete(ctx context.Context, path string) (*http.Response, error) {
	return c.do(ctx, http.MethodDelete, path, nil, nil)
}

func (c *ExtendedClient) put(ctx context.Context, path string, payload interface{}, headers map[string]string) (*http.Response, error) {
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		return c.do(ctx, http.MethodPut, path, bytes.NewBuffer(data), headers)
	}
	return c.do(ctx, http.MethodPut, path, nil, headers)
}

func (c *ExtendedClient) post(ctx context.Context, path string, payload interface{}, headers map[string]string) (*http.Response, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return c.do(ctx, http.MethodPost, path, bytes.NewBuffer(data), headers)
}

func (c *ExtendedClient) get(ctx context.Context, path string) (*http.Response, error) {
	return c.do(ctx, http.MethodGet, path, nil, nil)
}

// needed where pagerduty use a different endpoint for certain actions (eg: v2 events)
func (c *ExtendedClient) doWithEndpoint(ctx context.Context, endpoint, method, path string, authRequired bool, body io.Reader, headers map[string]string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, endpoint+path, body)
	if err != nil {
		return nil, fmt.Errorf("failed to build request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.pagerduty+json;version=2")

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	if authRequired {
		switch c.authType {
		case oauthToken:
			req.Header.Set("Authorization", "Bearer "+c.authToken)
		default:
			req.Header.Set("Authorization", "Token token="+c.authToken)
		}
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	return c.checkResponse(resp, err)
}

func (c *ExtendedClient) do(ctx context.Context, method, path string, body io.Reader, headers map[string]string) (*http.Response, error) {
	return c.doWithEndpoint(ctx, c.apiEndpoint, method, path, true, body, headers)
}

func (c *ExtendedClient) decodeJSON(resp *http.Response, payload interface{}) error {
	defer func() { _ = resp.Body.Close() }() // explicitly discard error

	decoder := json.NewDecoder(resp.Body)
	return decoder.Decode(payload)
}

func (c *ExtendedClient) checkResponse(resp *http.Response, err error) (*http.Response, error) {
	if err != nil {
		return resp, fmt.Errorf("Error calling the API endpoint: %v", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return resp, c.getErrorFromResponse(resp)
	}

	return resp, nil
}

func (c *ExtendedClient) getErrorFromResponse(resp *http.Response) pagerduty.APIError {
	// check whether the error response is declared as JSON
	if !strings.HasPrefix(resp.Header.Get("Content-Type"), "application/json") {
		aerr := pagerduty.APIError{
			StatusCode: resp.StatusCode,
		}

		return aerr
	}

	var document pagerduty.APIError

	// because of above check this probably won't fail, but it's possible...
	if err := c.decodeJSON(resp, &document); err != nil {
		aerr := pagerduty.APIError{
			StatusCode: resp.StatusCode,
		}

		return aerr
	}

	document.StatusCode = resp.StatusCode

	return document
}

// Helper function to determine wither additional parameters should use ? or & to append args
func getBasePrefix(basePath string) string {
	if strings.Contains(path.Base(basePath), "?") {
		return basePath + "&"
	}
	return basePath + "?"
}

// responseHandler is capable of parsing a response. At a minimum it must
// extract the page information for the current page. It can also execute
// additional necessary handling; for example, if a closure, it has access
// to the scope in which it was defined, and can be used to append data to
// a specific slice. The responseHandler is responsible for closing the response.
type responseHandler func(response *http.Response) (pagerduty.APIListObject, error)

func (c *ExtendedClient) pagedGet(ctx context.Context, basePath string, handler responseHandler) error {
	// Indicates whether there are still additional pages associated with request.
	var stillMore bool

	// Offset to set for the next page request.
	var nextOffset uint

	basePrefix := getBasePrefix(basePath)
	// While there are more pages, keep adjusting the offset to get all results.
	for stillMore, nextOffset = true, 0; stillMore; {
		response, err := c.do(ctx, http.MethodGet, fmt.Sprintf("%soffset=%d", basePrefix, nextOffset), nil, nil)
		if err != nil {
			return err
		}

		// Call handler to extract page information and execute additional necessary handling.
		pageInfo, err := handler(response)
		if err != nil {
			return err
		}

		// Bump the offset as necessary and set whether more results exist.
		nextOffset = pageInfo.Offset + pageInfo.Limit
		stillMore = pageInfo.More
	}

	return nil
}
