package binance

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	mainnetREST = "https://fapi.binance.com"
	testnetREST = "https://testnet.binancefuture.com"
)

// Client is a thin USD-M Futures REST client (Agent-side only).
type Client struct {
	baseURL string
	key     string
	secret  string

	httpc *http.Client

	muOffset sync.RWMutex
	offsetMs int64

	symMu sync.Mutex
	lot   map[string]float64 // symbol -> rounded stepSize (parsed float)
}

type ClientOptions struct {
	HTTPClient *http.Client
}

// NewUSDMMClient constructs a Futures client. apiKey/apiSecret must be loaded from env by the caller (never SaaS YAML).
func NewUSDMMClient(baseURL, apiKey, apiSecret string, opts *ClientOptions) *Client {
	httpc := http.DefaultClient
	if opts != nil && opts.HTTPClient != nil {
		httpc = opts.HTTPClient
	}
	if strings.TrimSpace(baseURL) == "" {
		baseURL = mainnetREST
	}
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		key:     apiKey,
		secret:  apiSecret,
		httpc:   httpc,
		lot:     map[string]float64{},
	}
}

// RESTBaseForConfig maps config flag to default REST host.
func RESTBaseForConfig(useTestnet bool) string {
	if useTestnet {
		return testnetREST
	}
	return mainnetREST
}

func (c *Client) SyncServerTime(ctx context.Context) error {
	raw, _, err := c.doPublicGET(ctx, "/fapi/v1/time", url.Values{})
	if err != nil {
		return err
	}
	var t struct {
		ServerTime int64 `json:"serverTime"`
	}
	if err := json.Unmarshal(raw, &t); err != nil {
		return fmt.Errorf("binance: parse time: %w", err)
	}
	local := time.Now().UnixMilli()
	c.muOffset.Lock()
	c.offsetMs = t.ServerTime - local
	c.muOffset.Unlock()
	return nil
}

func (c *Client) signedTimestamp() int64 {
	c.muOffset.RLock()
	off := c.offsetMs
	c.muOffset.RUnlock()
	return time.Now().UnixMilli() + off
}

func (c *Client) doPublicGET(ctx context.Context, path string, q url.Values) ([]byte, int, error) {
	u := c.baseURL + path
	if len(q) > 0 {
		u += "?" + q.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, 0, err
	}
	resp, err := c.httpc.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return b, resp.StatusCode, ParseAPIErrorFromResponse(resp.StatusCode, b, resp.Header)
	}
	return b, resp.StatusCode, nil
}

func (c *Client) doSignedGET(ctx context.Context, path string, params url.Values) ([]byte, int, error) {
	return c.doSigned(ctx, http.MethodGet, path, params)
}

func (c *Client) doSignedPOST(ctx context.Context, path string, params url.Values) ([]byte, int, error) {
	return c.doSigned(ctx, http.MethodPost, path, params)
}

func (c *Client) doSignedDELETE(ctx context.Context, path string, params url.Values) ([]byte, int, error) {
	return c.doSigned(ctx, http.MethodDelete, path, params)
}

func (c *Client) doSigned(ctx context.Context, method, path string, params url.Values) ([]byte, int, error) {
	if c.key == "" {
		return nil, 0, fmt.Errorf("binance: api key missing")
	}
	if params == nil {
		params = url.Values{}
	}
	params.Set("timestamp", strconv.FormatInt(c.signedTimestamp(), 10))
	if params.Get("recvWindow") == "" {
		params.Set("recvWindow", "5000")
	}
	query, sig, err := buildSignedQuery(c.secret, params)
	if err != nil {
		return nil, 0, err
	}
	fullURL := c.baseURL + path
	var req *http.Request
	switch method {
	case http.MethodGet, http.MethodDelete:
		req, err = http.NewRequestWithContext(ctx, method, fullURL+"?"+query+"&signature="+sig, nil)
	case http.MethodPost:
		req, err = http.NewRequestWithContext(ctx, method, fullURL,
			strings.NewReader(query+"&signature="+sig))
	default:
		return nil, 0, fmt.Errorf("binance: unsupported method %s", method)
	}
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("X-MBX-APIKEY", c.key)
	if method == http.MethodPost {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	resp, err := c.httpc.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		apiErr := ParseAPIErrorFromResponse(resp.StatusCode, b, resp.Header)
		var ep *APIError
		if AsAPIError(apiErr, &ep) && (ep.RetryReason == "resync_timestamp" || ep.Code == -1021) {
			_ = c.SyncServerTime(ctx)
		}
		return b, resp.StatusCode, apiErr
	}
	return b, resp.StatusCode, nil
}
