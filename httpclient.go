package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

//Client used to communicate with Cloud Foundry
type OauthClient struct {
	config     *Config
	Endpoint   *oauth2.Endpoint
	HttpClient *http.Client
}

//Config is used to configure the creation of a client
type Config struct {
	ApiAddress        string `required:"true"`
	ClientID          string `required:"true"`
	ClientSecret      string `required:"true"`
	GrantType         string
	SkipSslValidation bool
	Token             string
	TokenSource       oauth2.TokenSource
}

// request is used to help build up a request
type request struct {
	method string
	url    string
	params url.Values
	header *http.Header
	body   io.Reader
	obj    interface{}
}

// NewClient returns a new client
func NewOauthClient(config *Config) (client *OauthClient, err error) {
	ctx := oauth2.NoContext
	httpClient := http.DefaultClient
	if config.SkipSslValidation {
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		httpClient = &http.Client{Transport: tr}
	}

	ctx = context.WithValue(ctx, oauth2.HTTPClient, httpClient)

	endpoint := oauth2.Endpoint{
		AuthURL:  config.ApiAddress + "/oauth/auth",
		TokenURL: config.ApiAddress + "/oauth/token",
	}

	authConfig := &clientcredentials.Config{
		ClientID:     config.ClientID,
		ClientSecret: config.ClientSecret,
		Scopes:       []string{""},
		TokenURL:     endpoint.TokenURL,
	}

	config.TokenSource = authConfig.TokenSource(ctx)

	return &OauthClient{
		config:     config,
		Endpoint:   &endpoint,
		HttpClient: authConfig.Client(ctx),
	}, nil
}

// NewRequest is used to create a new request
func (c *OauthClient) NewRequest(method, path string) *request {
	r := &request{
		method: method,
		url:    c.config.ApiAddress + path,
		params: make(map[string][]string),
	}
	return r
}

// DoRequest runs a request with our client
func (c *OauthClient) DoRequest(r *request) (*http.Response, error) {
	req, err := r.toHTTP()
	if err != nil {
		return nil, err
	}
	resp, err := c.HttpClient.Do(req)
	return resp, err
}

// toHTTP converts the request to an HTTP request
func (r *request) toHTTP() (*http.Request, error) {
	// Check if we should encode the body
	if r.body == nil && r.obj != nil {
		b, err := encodeBody(r.obj)
		if err != nil {
			return nil, err
		}
		r.body = b
	}

	// Create the HTTP request

	req, err := http.NewRequest(r.method, r.url, r.body)
	if err != nil {
		return req, err
	}

	req.Header = *r.header

	return req, err
}

// decodeBody is used to JSON decode a body
func decodeBody(resp *http.Response, out interface{}) error {
	defer resp.Body.Close()
	dec := json.NewDecoder(resp.Body)
	return dec.Decode(out)
}

// encodeBody is used to encode a request body
func encodeBody(obj interface{}) (io.Reader, error) {
	buf := bytes.NewBuffer(nil)
	enc := json.NewEncoder(buf)
	if err := enc.Encode(obj); err != nil {
		return nil, err
	}
	return buf, nil
}

func (c *OauthClient) GetToken() (string, error) {
	token, err := c.config.TokenSource.Token()
	if err != nil {
		return "", fmt.Errorf("Error getting bearer token: %v", err)
	}
	return "bearer " + token.AccessToken, nil
}
