package mirror

// AWS Signature Version 4, reduced to what read-only GET requests against an
// S3-compatible endpoint need (Cloudflare R2 in particular). Implemented on
// the standard library to keep the project dependency-light.
//
// Reference: "Authenticating Requests: Using the Authorization Header (AWS
// Signature Version 4)" in the S3 API docs.

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

const (
	sigv4Algorithm  = "AWS4-HMAC-SHA256"
	sigv4Request    = "aws4_request"
	r2DefaultRegion = "auto"
	// sha256 of the empty body; GET requests carry no payload.
	emptyPayloadSHA = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
)

// sigv4Signer signs requests with long-term credentials via the
// Authorization header. R2 accepts any region string; "auto" is the
// convention, but the field exists so tests can verify against AWS vectors.
type sigv4Signer struct {
	accessKeyID     string
	secretAccessKey string
	region          string
	service         string
}

// sign adds x-amz-date, x-amz-content-sha256 and Authorization headers to
// req. payloadHash is the hex sha256 of the request body (emptyPayloadSHA
// for a GET). extraHeaders names additional request headers (lowercase,
// e.g. "range") to cover in the signature.
func (s sigv4Signer) sign(req *http.Request, payloadHash string, now time.Time, extraHeaders ...string) {
	if s.region == "" {
		s.region = r2DefaultRegion
	}
	if s.service == "" {
		s.service = "s3"
	}
	amzDate := now.UTC().Format("20060102T150405Z")
	dateStamp := now.UTC().Format("20060102")

	req.Header.Set("x-amz-date", amzDate)
	req.Header.Set("x-amz-content-sha256", payloadHash)

	host := req.Host
	if host == "" {
		host = req.URL.Host
	}

	names := append([]string{"host", "x-amz-content-sha256", "x-amz-date"}, extraHeaders...)
	names = sortedUnique(names)

	var canonicalHeaders strings.Builder
	for _, name := range names {
		value := ""
		switch name {
		case "host":
			value = host
		case "x-amz-content-sha256":
			value = payloadHash
		case "x-amz-date":
			value = amzDate
		default:
			value = canonicalHeaderValue(req.Header.Get(name))
		}
		canonicalHeaders.WriteString(name)
		canonicalHeaders.WriteByte(':')
		canonicalHeaders.WriteString(value)
		canonicalHeaders.WriteByte('\n')
	}
	signedHeaders := strings.Join(names, ";")

	// S3 canonical URIs are the URI-encoded resource path, not double-encoded.
	// We build request URLs pre-encoded, so EscapedPath is the canonical URI.
	canonicalURI := req.URL.EscapedPath()
	if canonicalURI == "" {
		canonicalURI = "/"
	}

	canonicalRequest := strings.Join([]string{
		req.Method,
		canonicalURI,
		canonicalQuery(req.URL),
		canonicalHeaders.String(),
		signedHeaders,
		payloadHash,
	}, "\n")

	scope := strings.Join([]string{dateStamp, s.region, s.service, sigv4Request}, "/")
	stringToSign := strings.Join([]string{
		sigv4Algorithm,
		amzDate,
		scope,
		sha256Hex([]byte(canonicalRequest)),
	}, "\n")

	signingKey := hmacSHA256(
		hmacSHA256(
			hmacSHA256(
				hmacSHA256([]byte("AWS4"+s.secretAccessKey), []byte(dateStamp)),
				[]byte(s.region)),
			[]byte(s.service)),
		[]byte(sigv4Request))
	signature := hex.EncodeToString(hmacSHA256(signingKey, []byte(stringToSign)))

	req.Header.Set("Authorization", fmt.Sprintf(
		"%s Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		sigv4Algorithm, s.accessKeyID, scope, signedHeaders, signature))
}

func canonicalQuery(u *url.URL) string {
	if u.RawQuery == "" {
		return ""
	}
	values := u.Query()
	keys := make([]string, 0, len(values))
	for k := range values {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var parts []string
	for _, k := range keys {
		vs := values[k]
		sort.Strings(vs)
		for _, v := range vs {
			parts = append(parts, awsURIEncode(k, true)+"="+awsURIEncode(v, true))
		}
	}
	return strings.Join(parts, "&")
}

// canonicalHeaderValue trims surrounding whitespace and collapses internal
// sequential spaces, per the SigV4 canonical-headers rules.
func canonicalHeaderValue(v string) string {
	fields := strings.Fields(v)
	return strings.Join(fields, " ")
}

// awsURIEncode percent-encodes everything except unreserved characters
// (A-Z a-z 0-9 - _ . ~); slashes are kept literal unless encodeSlash is set.
func awsURIEncode(s string, encodeSlash bool) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c >= 'A' && c <= 'Z', c >= 'a' && c <= 'z', c >= '0' && c <= '9',
			c == '-', c == '_', c == '.', c == '~',
			c == '/' && !encodeSlash:
			b.WriteByte(c)
		default:
			fmt.Fprintf(&b, "%%%02X", c)
		}
	}
	return b.String()
}

func sortedUnique(names []string) []string {
	sort.Strings(names)
	out := names[:0]
	var prev string
	for i, n := range names {
		if i > 0 && n == prev {
			continue
		}
		prev = n
		out = append(out, n)
	}
	return out
}

func hmacSHA256(key, data []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return h.Sum(nil)
}

func sha256Hex(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}
