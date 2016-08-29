package oauth

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
type Client struct {
	config      *ClientConfig
	Endpoint    *oauth2.Endpoint
	HttpClient  *http.Client
	TokenSource oauth2.TokenSource
}

//Config is used to configure the creation of a client
type ClientConfig struct {
	ApiAddress        string `required:"true"`
	ClientID          string `required:"true"`
	ClientSecret      string `required:"true"`
	GrantType         string
	SkipSslValidation bool
}

// Request is used to help build up a request
type Request struct {
	method string
	url    string
	params url.Values
	header http.Header
	body   io.Reader
	obj    interface{}
}

func (r *Request) AddHeader(key, value string) {
	r.header.Add(key, value)
}

func (r *Request) AddParam(key, value string) {
	r.params.Add(key, value)
}

func (r *Request) SetBody(body io.Reader) {
	r.body = body
}

func (r *Request) SetPayload(obj interface{}) {
	r.obj = obj
}

// NewClient returns a new client
func NewClient(config *ClientConfig) (client *Client, err error) {
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
	return &Client{
		config:      config,
		Endpoint:    &endpoint,
		HttpClient:  authConfig.Client(ctx),
		TokenSource: authConfig.TokenSource(ctx),
	}, nil
}

// NewRequest is used to create a new request
func (c *Client) NewRequest(method, path string) *Request {
	r := &Request{
		method: method,
		url:    c.config.ApiAddress + path,
		params: make(map[string][]string),
		header: make(map[string][]string),
	}
	return r
}

// DoRequest runs a request with our client
func (c *Client) DoRequest(r *Request) (*http.Response, error) {
	req, err := r.toHTTP()
	if err != nil {
		return nil, err
	}
	resp, err := c.HttpClient.Do(req)
	return resp, err
}

// toHTTP converts the Request to an HTTP request
func (r *Request) toHTTP() (*http.Request, error) {
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

	req.Header = r.header

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

func (c *Client) GetToken() (string, error) {
	token, err := c.TokenSource.Token()
	if err != nil {
		return "", fmt.Errorf("Error getting bearer token: %v", err)
	}
	return "bearer " + token.AccessToken, nil
}
