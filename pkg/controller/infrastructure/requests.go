package infrastructure

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	dodasv1alpha1 "github.com/dodas-ts/dodas-operator/pkg/apis/dodas/v1alpha1"
)

// Request input struct
type Request struct {
	URL         string
	RequestType string
	Headers     map[string]string
	AuthUser    string
	AuthPwd     string
	Content     []byte
	Timeout     time.Duration
}

func validateRequest(r Request) (Request, error) {

	validatedRequest := r

	if &r.Timeout == nil {
		validatedRequest.Timeout = 5 * time.Minute
	}

	if r.URL == "" {
		return Request{}, fmt.Errorf("URL not specified")
	}

	if r.RequestType == "" {
		validatedRequest.RequestType = "GET"
	}

	return validatedRequest, nil
}

// RefreshToken wraps actions for token refreshing
func RefreshToken(refreshToken string, clientConf *dodasv1alpha1.Infrastructure) (string, error) {

	var token string

	clientID := clientConf.Spec.AllowRefresh.ClientID
	clientSecret := clientConf.Spec.AllowRefresh.ClientSecret
	IAMTokenEndpoint := clientConf.Spec.AllowRefresh.IAMTokenEndpoint

	client := &http.Client{
		Timeout: 300 * time.Second,
	}

	req, _ := http.NewRequest("GET", IAMTokenEndpoint, nil)

	req.SetBasicAuth(clientID, clientSecret)

	req.Header.Set("grant_type", "refresh_token")
	req.Header.Set("refresh_token", refreshToken)

	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}

	body, _ := ioutil.ReadAll(resp.Body)

	if resp.StatusCode == 200 {

		type accessTokenStruct struct {
			AccessToken string `json:"access_token"`
		}

		var accessTokenJSON accessTokenStruct

		err = json.Unmarshal(body, &accessTokenJSON)
		if err != nil {
			return "", err
		}

		token = accessTokenJSON.AccessToken

	} else {
		return "", fmt.Errorf("ERROR: %s", string(body))
	}

	return token, nil
}

// MakeRequest function based on inputs
func MakeRequest(request Request) (body []byte, statusCode int, err error) {

	var req *http.Request

	r, err := validateRequest(request)
	if err != nil {
		return nil, -1, fmt.Errorf("Failed to validate request inputs %s", err)
	}

	client := &http.Client{
		Timeout: r.Timeout,
	}

	switch r.RequestType {
	case "POST":
		req, err = http.NewRequest(r.RequestType, r.URL, bytes.NewBuffer(r.Content))
		if err != nil {
			return nil, -1, fmt.Errorf("Failed to create POST http request: %s", err)
		}
	default:
		req, err = http.NewRequest(r.RequestType, r.URL, nil)
		if err != nil {
			return nil, -1, fmt.Errorf("Failed to create %s http request: %s", r.RequestType, err)
		}
	}

	for key, value := range r.Headers {
		req.Header.Set(key, value)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, -1, fmt.Errorf("Remote request failed: %s", err)
	}

	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, -1, fmt.Errorf("Failed to read the response: %s", err)
	}

	return body, resp.StatusCode, nil
}
