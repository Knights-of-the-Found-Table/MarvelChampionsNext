package mirror

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// The worked GET-object example from the AWS S3 SigV4 documentation
// ("Authenticating Requests: Using the Authorization Header"), including a
// Range header, verified against the published signature.
func TestSigv4AWSExampleVector(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "https://examplebucket.s3.amazonaws.com/test.txt", nil)
	req.Header.Set("Range", "bytes=0-9")
	signer := sigv4Signer{
		accessKeyID:     "AKIAIOSFODNN7EXAMPLE",
		secretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
		region:          "us-east-1",
		service:         "s3",
	}
	signer.sign(req, emptyPayloadSHA, time.Date(2013, 5, 24, 0, 0, 0, 0, time.UTC), "range")

	const want = "AWS4-HMAC-SHA256 " +
		"Credential=AKIAIOSFODNN7EXAMPLE/20130524/us-east-1/s3/aws4_request, " +
		"SignedHeaders=host;range;x-amz-content-sha256;x-amz-date, " +
		"Signature=f0e8bdb87c964420e857bd35b5d6ed310bd44f0170aba48dd91039c6036bdb41"
	if got := req.Header.Get("Authorization"); got != want {
		t.Fatalf("authorization:\n got  %s\n want %s", got, want)
	}
	if got := req.Header.Get("x-amz-date"); got != "20130524T000000Z" {
		t.Fatalf("x-amz-date = %q", got)
	}
	if got := req.Header.Get("x-amz-content-sha256"); got != emptyPayloadSHA {
		t.Fatalf("x-amz-content-sha256 = %q", got)
	}
}

// Default region must be "auto" (R2's convention) when unset.
func TestSigv4DefaultRegion(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:1/bucket/key.png", nil)
	signer := sigv4Signer{accessKeyID: "ak", secretAccessKey: "sk"}
	signer.sign(req, emptyPayloadSHA, time.Unix(0, 0).UTC())
	const wantScope = "/19700101/auto/s3/aws4_request"
	if got := req.Header.Get("Authorization"); !strings.Contains(got, wantScope) {
		t.Fatalf("authorization %q lacks scope %q", got, wantScope)
	}
}

func TestAwsURIEncode(t *testing.T) {
	cases := []struct {
		in, want string
		slash    bool
	}{
		{in: "/bundles/cards/01001a.png", want: "/bundles/cards/01001a.png"},
		{in: "a b/c+d", want: "a%20b/c%2Bd"},
		{in: "a b/c+d", want: "a%20b%2Fc%2Bd", slash: true},
		{in: "ünï", want: "%C3%BCn%C3%AF"},
	}
	for _, c := range cases {
		if got := awsURIEncode(c.in, c.slash); got != c.want {
			t.Errorf("awsURIEncode(%q, %v) = %q, want %q", c.in, c.slash, got, c.want)
		}
	}
}
