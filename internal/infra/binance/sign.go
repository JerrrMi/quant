package binance

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/url"
	"sort"
	"strings"
)

func signQuery(secret string, query string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = io.WriteString(mac, query)
	return hex.EncodeToString(mac.Sum(nil))
}

func buildSignedQuery(secret string, params url.Values) (query string, signature string, err error) {
	if secret == "" {
		return "", "", fmt.Errorf("binance: api secret required for signed request")
	}
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	for i, k := range keys {
		if i > 0 {
			b.WriteByte('&')
		}
		v := strings.TrimSpace(params.Get(k))
		b.WriteString(k)
		b.WriteByte('=')
		b.WriteString(url.QueryEscape(v))
	}
	raw := b.String()
	sig := signQuery(secret, raw)
	return raw, sig, nil
}
