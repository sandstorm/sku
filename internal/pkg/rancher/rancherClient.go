package rancher

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

type rancherClient struct {
	apiEndpointUrl      string
	token               string
	absoluteStoragePath string
	httpClient          *http.Client
}

// fetch an URL as given by relativeUrl, and return the body contents as byte array; already pretty printed.
func (rc *rancherClient) fetchUrl(relativeUrl string) *RancherApiResponse {
	req, _ := http.NewRequest("GET", joinUrl(rc.apiEndpointUrl, relativeUrl), nil)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", rc.token))
	resp, err := rc.httpClient.Do(req)

	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	responseBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	var response *RancherApiResponse
	err = json.Unmarshal(responseBody, &response)
	if err != nil {
		log.Fatal(err)
	}

	return response
}


type RancherApiResponse struct {
	Type string `json:"type"`

	// filled for lists of things, where Type == "collection"
	ResourceType string `json:"resourceType"`
	Links map[string]string `json:"links"`

	// filled for lists of things, where Type == "collection"
	// is an array of: maps from string to anything
	Data []map[string]interface{} `json:"data"`

	Pagination RancherApiPagination `json:"pagination"`
}

type RancherApiPagination struct {
	Limit int `json:"limit"`
	Total int `json:"total"`
}
