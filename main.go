package main

import (
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/kelseyhightower/envconfig"
)

func main() {
	log.SetOutput(os.Stdout)

	var config Config
	if err := envconfig.Process("uaa", &config); err != nil {
		log.Fatalf("Failed to process required environment variables: %v", err)
	}

	client, err := NewOauthClient(&config)
	if err != nil {
		log.Fatalf("Failed to initialize client: %v\n", err)
	}

	token, err := client.GetToken()
	if err != nil {
		log.Fatalf("Failed to get token; %v", err)
	}

	log.Printf("Token: %s\n", token)

	acceptHeader := &http.Header{}
	acceptHeader.Add("Accept", "application/json")

	executeRequest(client, "GET", "/info", acceptHeader)
	executeRequest(client, "GET", "/Users", acceptHeader)
}

func executeRequest(client *OauthClient, method string, path string, header *http.Header) {
	r := client.NewRequest(method, path)
	r.header = header
	log.Printf("executing request %vn", r)

	resp, err := client.DoRequest(r)
	if err != nil {
		log.Printf("Failed to execute request %v: %v\n", r, err)
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Failed to read response body: %v\n", err)
		return
	}

	log.Printf("Response: \n%s\n", string(body))
}
