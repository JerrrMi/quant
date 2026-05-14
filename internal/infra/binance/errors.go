package binance

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// APIError wraps a Binance Futures error payload with optional HTTP metadata.
type APIError struct {
	Code           int
	Msg            string
	HTTPStatus     int
	Retryable      bool
	RetryAfter     time.Duration
	RetryReason    string
	RequestID      string
	ResponseBody   string
}

func (e *APIError) Error() string {
	if e == nil {
		return "<nil APIError>"
	}
	return fmt.Sprintf("binance futures api: code=%d msg=%q http=%d", e.Code, e.Msg, e.HTTPStatus)
}

type errPayload struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

// ParseAPIErrorFromResponse builds an APIError when the body is JSON with code/msg.
func ParseAPIErrorFromResponse(status int, body []byte, hdr http.Header) *APIError {
	var p errPayload
	if err := json.Unmarshal(body, &p); err != nil || p.Code == 0 && p.Msg == "" {
		msg := strings.TrimSpace(string(body))
		if len(msg) > 240 {
			msg = msg[:240] + "…"
		}
		return &APIError{
			Code:        0,
			Msg:         msg,
			HTTPStatus:  status,
			Retryable:   status == http.StatusTooManyRequests || status >= 500,
			RetryReason: classifyHTTPRetry(status, hdr),
			RequestID:   hdr.Get("X-Mbx-Uuid"),
			ResponseBody: shorten(string(body), 512),
		}
	}
	api := &APIError{
		Code:         p.Code,
		Msg:          p.Msg,
		HTTPStatus:   status,
		RequestID:    hdr.Get("X-Mbx-Uuid"),
		ResponseBody: shorten(string(body), 512),
	}
	api.applyCodeHeuristics(hdr)
	return api
}

func classifyHTTPRetry(status int, hdr http.Header) string {
	switch {
	case status == 418:
		return "ip_ban_or_waf"
	case status == 429 || status == http.StatusTooManyRequests:
		return "rate_limit_http"
	case status >= 500:
		return "server_error"
	default:
		return ""
	}
}

func (e *APIError) applyCodeHeuristics(hdr http.Header) {
	// Reference: public Binance Futures error code list (selected).
	switch e.Code {
	case -1003: // too many requests / way too many requests
		e.Retryable = true
		e.RetryReason = "rate_limit"
		e.RetryAfter = parseRetryAfter(hdr, 2*time.Second)
	case -1021, -1022: // timestamp / signature
		e.Retryable = true
		e.RetryReason = "resync_timestamp"
		e.RetryAfter = 200 * time.Millisecond
	case -5022, -5021: // collateral / margin issues — not solved by blind retry
		e.Retryable = false
	default:
		if e.HTTPStatus >= 500 {
			e.Retryable = true
			e.RetryReason = "http_5xx"
			e.RetryAfter = 500 * time.Millisecond
		}
	}
}

func parseRetryAfter(hdr http.Header, def time.Duration) time.Duration {
	v := strings.TrimSpace(hdr.Get("Retry-After"))
	if v == "" {
		return def
	}
	if sec, err := strconv.Atoi(v); err == nil && sec > 0 {
		return time.Duration(sec) * time.Second
	}
	return def
}

func shorten(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

// IsDuplicateClientOrder reports whether the venue likely rejected because the client order id was reused.
func IsDuplicateClientOrder(err error) bool {
	var api *APIError
	if !AsAPIError(err, &api) {
		return false
	}
	// Binance may vary message text; match substrings defensively.
	msg := strings.ToLower(api.Msg)
	if strings.Contains(msg, "duplicate") && strings.Contains(msg, "order") {
		return true
	}
	if strings.Contains(msg, "client order id") && strings.Contains(msg, "exist") {
		return true
	}
	return false
}

// AsAPIError unwraps APIError from err via errors.As.
func AsAPIError(err error, target **APIError) bool {
	if err == nil || target == nil {
		return false
	}
	var a *APIError
	if errors.As(err, &a) {
		*target = a
		return true
	}
	return false
}
