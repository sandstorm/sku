package rancher

import (
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"reflect"
	"regexp"
)

func RunBackup(apiEndpointUrl string, token string, absoluteStoragePath string) {

	rc := &rancherClient{
		apiEndpointUrl:      apiEndpointUrl,
		token:               token,
		absoluteStoragePath: absoluteStoragePath,
		httpClient: &http.Client{
		},
	}

	// iterate through the root API - all "links" existing.
	apiRoot := rc.fetchUrl("")
	for k, collectionUrl := range apiRoot.Links {
		// We do not need self or root
		// we skip clusterRegistrationTokens as they contain very sensitive information
		// WORKAROUND: we do not need templates and templateversions (which are huge); because they are simply cached from the remote catalog.
		// WORKAROUND: LDAP configs are not part of our cluster here.
		if k == "self" || k == "root" || k == "subscribe"|| k == "clusterRegistrationTokens" || k == "templates" || k == "templateVersions" || k == "ldapConfigs" {
			continue
		}

		fetchCollection(collectionUrl, absoluteStoragePath, rc, func(el collectionElement) {
			if k == "clusters" {
				log.Printf("-- !! extracting StorageClasses")
				storageClassesUrl := extractLinkTargetUrl(el, "storageClasses")
				fetchCollection(storageClassesUrl, path.Join(absoluteStoragePath, "_cluster_" + extractId(el, collectionUrl)), rc, nil)
			}

			if k == "projects" {
				log.Printf("-- !! extracting Project details")

				for k, nestedCollectionUrl := range extractLinks(el) {
					// we won't dump secrets or namespacedSecrets because of their sensitive nature
					if k == "self" || k == "remove" || k == "update" || k == "subscribe" || k == "secrets" || k == "namespacedSecrets" {
						continue
					}
						fetchCollection(nestedCollectionUrl, path.Join(absoluteStoragePath, "_project_" + extractId(el, collectionUrl)), rc, nil)
				}
			}

		})
	}
}


type collectionElement map[string]interface{}

func fetchCollection(collectionUrl string, absoluteStoragePath string, rc *rancherClient, elementPostProcessCallback func(element collectionElement)) {
	log.Printf("- %s\n", collectionUrl)
	collection := rc.fetchUrl(collectionUrl)
	ensureValidCollection(collection, collectionUrl)

	// create empty collection storage path
	collectionStoragePath := path.Join(absoluteStoragePath, collection.ResourceType)
	ensurePathExistsAndIsEmpty(collectionStoragePath)

	// dump the individual properties to files.
	for _, element := range collection.Data {
		id := extractId(element, collectionUrl)

		err := ioutil.WriteFile(path.Join(collectionStoragePath, sanitizeStringForFile(id) + ".json"), prettyPrintJsonString(element), 0644)
		if err != nil {
			log.Fatal(err)
		}

		if elementPostProcessCallback != nil {
			elementPostProcessCallback(element)
		}
	}
}

var validFilenamesRegexp, _ = regexp.Compile("[^a-zA-Z0-9-_:.]")

func sanitizeStringForFile(s string) string {
	return validFilenamesRegexp.ReplaceAllString(s, "_")
}

func extractId(element map[string]interface{}, url string) string {
	id, ok := element["id"].(string)
	if !ok {
		log.Fatalf("ID could not be extracted for %v", element)
	}
	if len(id) == 0 {
		log.Fatalf("ID was null at URL %s", url)
	}

	return id
}


func extractLinks(element collectionElement) map[string]string {
	if element["links"] == nil {
		log.Fatalf("links key not found in Element %v", element)
	}
	links, ok := element["links"].(map[string]interface{})
	if !ok {
		log.Fatalf("links could not be extracted, target type was %s for %v", reflect.TypeOf(element["links"]), element)
	}

	result := make(map[string]string)

	for k, v := range links {
		result[k] = v.(string)
	}

	return result
}

func extractLinkTargetUrl(element collectionElement, linkName string) string {
	links := extractLinks(element)
	if len(links[linkName]) == 0 {
		log.Fatalf("link %s could not be extracted in element %v", linkName, links)
	}

	return links[linkName]
}

func ensurePathExistsAndIsEmpty(path string) {
	err := os.RemoveAll(path)
	if err != nil {
		log.Fatal(err)
	}

	err = os.MkdirAll(path, 0755)
	if err != nil {
		log.Fatal(err)
	}
}

func ensureValidCollection(collection *RancherApiResponse, url string) {
	if collection.Type != "collection" {
		log.Fatalf("The rancher API URL: %s was not of type collection, found %s", url, collection.Type)
	}

	if collection.Pagination.Limit == collection.Pagination.Total {
		log.Fatalf("We would need to paginate for URL %s, which we do not support yet. Needs to be implemented!", url)
	}

	if len(collection.ResourceType) == 0 {
		log.Fatalf("The rancher API URL %s did not have a ResourceType set", url)
	}

}
