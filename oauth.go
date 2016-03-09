package cloudsight

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	urlPkg "net/url"
	"strings"
	"time"
)

const (
	oauthSignatureMethod = "HMAC-SHA1"
	oauthVersion         = "1.0"
)

func oauthSign(method, url, key, secret string, params Params) (string, error) {
	if params == nil {
		params = Params{}
	}

	// Get random nonce
	nonceBuf := make([]byte, 20)
	if _, err := rand.Read(nonceBuf); err != nil {
		return "", err
	}
	hash := sha256.Sum256(nonceBuf)
	params["oauth_nonce"] = hex.EncodeToString(hash[:])

	params["oauth_consumer_key"] = key
	params["oauth_signature_method"] = oauthSignatureMethod
	params["oauth_timestamp"] = fmt.Sprint(time.Now().Unix())
	params["oauth_version"] = oauthVersion

	baseString := strings.Join([]string{
		strings.ToUpper(method),
		urlPkg.QueryEscape(url),
		urlPkg.QueryEscape(params.values().Encode()),
	}, "&")

	secret = fmt.Sprintf("%s&", urlPkg.QueryEscape(secret))

	h := hmac.New(sha1.New, []byte(secret))
	h.Write([]byte(baseString))
	signature := h.Sum(nil)

	headerValues := []string{
		fmt.Sprintf("oauth_consumer_key=\"%s\"", params["oauth_consumer_key"]),
		fmt.Sprintf("oauth_nonce=\"%s\"", params["oauth_nonce"]),
		fmt.Sprintf("oauth_signature=\"%s\"", base64.StdEncoding.EncodeToString(signature)),
		fmt.Sprintf("oauth_signature_method=\"%s\"", oauthSignatureMethod),
		fmt.Sprintf("oauth_timestamp=\"%s\"", params["oauth_timestamp"]),
		fmt.Sprintf("oauth_version=\"%s\"", oauthVersion),
	}

	return fmt.Sprintf("OAuth %s", strings.Join(headerValues, ", ")), nil
}
