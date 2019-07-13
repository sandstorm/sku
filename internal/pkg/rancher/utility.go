package rancher

import (
	"encoding/json"
	"log"
	"net/url"
)

func joinUrl(baseUrl, relativeUrl string) string {
	relativeUrlParsed, err := url.Parse(relativeUrl)
	if err != nil {
		log.Fatal(err)
	}
	base, err := url.Parse(baseUrl)
	if err != nil {
		log.Fatal(err)
	}
	u := base.ResolveReference(relativeUrlParsed)

	return u.String()
}

func prettyPrintJsonString(element map[string]interface{}) []byte {
	result, err := json.MarshalIndent(element, "", "  ")
	if err != nil {
		log.Fatal(err)
	}

	return result
}
