package restore

import (
	"fmt"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// https://github.com/vmware-tanzu/velero/blob/3b2e9036d178831e9be9aa90403c4aad42793cb6/pkg/restore/restore.go
// https://github.com/vmware-tanzu/velero/blob/3b2e9036d178831e9be9aa90403c4aad42793cb6/pkg/restore/restore.go

type KubeFile struct {
	Parsed       ParsedKubernetesManifestParts
	Path         string
	FullKubeFile map[string]interface{}
	SkipReasons  []string
}

type ParsedKubernetesManifestParts struct {
	Kind       string `yaml:"kind"`
	ApiVersion string `yaml:"apiVersion"`
	Metadata   struct {
		Name            string            `yaml:"name"`
		Namespace       string            `yaml:"namespace"`
		Labels          map[string]string `yaml:"labels"`
		Annotations     map[string]string `yaml:"annotations"`
		OwnerReferences []struct {
			Kind       string `yaml:"kind"`
			ApiVersion string `yaml:"apiVersion"`
			Name       string `yaml:"name"`
		} `yaml:"ownerReferences"`
	} `yaml:"metadata"`

	// for Kind == Secret
	Secrets []struct {
		Name string `yaml:"name"`
	} `yaml:"secrets"`
	Type string `yaml:"type"`

	// for Kind == ClusterRoleBinding
	RoleRef struct {
		Kind     string `yaml:"kind"`
		ApiGroup string `yaml:"apiGroup"`
		Name     string `yaml:"name"`
	} `yaml:"roleRef"`
	Subjects []struct {
		Kind      string `yaml:"kind"`
		Name      string `yaml:"name"`
		Namespace string `yaml:"namespace"`
	} `yaml:"subjects"`
}

//  rg --files-without-match 'ownerRefer|norman'
//
//  - ignore endpoints which have the same name as a service (no matter whether service is filtered or not)
//  - ignore stuff with ownerReference
//  - ignore stuff with "norman" creator
//  - ignore default serviceaccount
//  - for every serviceaccount, check global policy...
//  - for every CRD, check whether CRD exists
//  - for secret, ignore if that's a service account secret
func BuildCleanManifestsCommand() *cobra.Command {
	var filename string = ""

	cleanManifestsCommand := &cobra.Command{
		Use:   "clean-manifests",
		Short: "ALPHA: (sandstorm) Filter and clean kubernetes manifests; making them ready to apply them to a new cluster (during migration)",
		Long: `
ALPHA Quality!

Filters out all resources which are managed by another entity, or which are managed by a K8S controller. This means
the output of this command is ready to be used during a restore-from-backup operation.

NOTE: If you are deploying an operator, you MANUALLY need to apply the CustomResourceDefinition beforehand!
      This is needed because we have no way to detect automatically which CRD is handled by the current operator (in
      fact, we don't even know if a Deployment is an "operator" or not, as this is a conceptual thing.
`,
		Example: `
		# 1) CREATE THE NAMESPACE and switch into it
		kubectl create namespace your-namespace-name
		sku ns your-namespace-name

		# 2) (optional) IMPORT A CUSTOM RESOURCE DEFINITION
		sku backup-restore clean-manifests -f ../../GLOBAL/config/CustomResourceDefinition-.....yaml
		# now, validate that the result looks good, then apply it.
		sku backup-restore clean-manifests -f ../../GLOBAL/config/CustomResourceDefinition-.....yaml | kubectl apply -f -

		# 3) IMPORT THE RESOURCES
		sku backup-restore clean-manifests -f .
		# now, validate that the result looks good, then apply it.
		sku backup-restore clean-manifests -f . | kubectl apply -f - --dry-run=client
		sku backup-restore clean-manifests -f . | kubectl apply -f -
`,

		// TODO: namespace remapping!
		Args: cobra.ExactArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			if len(filename) == 0 {
				log.Fatal("filename must be given")
			}

			fileList := buildFileListToRead(filename)
			kubeFiles := readKubeFiles(fileList)
			kubeFiles = filterKubeFiles(kubeFiles)
			kubeFiles = addExtraGlobalKubeFiles(kubeFiles)
			kubeFiles = cleanManifests(kubeFiles)
			kubeFiles = cleanManifestsTypeSpecific(kubeFiles)

			for _, kubeFile := range kubeFiles {
				if len(kubeFile.SkipReasons) > 0 {
					fmt.Fprintf(os.Stderr, "- Skipping %s\n", kubeFile.Path)
					for _, skipReason := range kubeFile.SkipReasons {
						fmt.Fprintf(os.Stderr, "    - %s\n", skipReason)
					}
				} else {
					fmt.Fprintf(os.Stdout, "---\n")

					fullKubeFile, err := yaml.Marshal(&kubeFile.FullKubeFile)
					if err != nil {
						log.Fatalf("could not create YAML for %s: %s", kubeFile.Path, err)
					}
					fmt.Fprintf(os.Stdout, "%s\n", string(fullKubeFile))
				}
			}
		},
	}

	cleanManifestsCommand.Flags().StringVarP(&filename, "filename", "f", "", "filename that contains the configuration to apply")

	return cleanManifestsCommand
}

func readKubeFiles(fileList []string) []*KubeFile {
	kubeFiles := make([]*KubeFile, 0, 0)
	for _, fileName := range fileList {
		fileNameBytes, err := ioutil.ReadFile(fileName)
		if err != nil {
			log.Fatalf("File %s could not be read: %s", fileName, err)
		}
		kubeFile := &KubeFile{
			Path: fileName,
		}
		err = yaml.Unmarshal(fileNameBytes, &(kubeFile.Parsed))
		yaml.Unmarshal(fileNameBytes, &(kubeFile.FullKubeFile))
		if err != nil {
			log.Fatalf("File %s could not be YAML-parsed: %s", fileName, err)
		}

		kubeFiles = append(kubeFiles, kubeFile)
	}

	return kubeFiles
}

func filterKubeFiles(kubeFiles []*KubeFile) []*KubeFile {
	for _, kubeFile := range kubeFiles {
		if len(kubeFile.Parsed.Metadata.Labels["authz.cluster.cattle.io/rtb-owner-updated"]) > 0 {
			kubeFile.SkipReasons = append(kubeFile.SkipReasons, "auto-created from Rancher - label authz.cluster.cattle.io/rtb-owner-updated")
		}

		if len(kubeFile.Parsed.Metadata.OwnerReferences) > 0 {
			kubeFile.SkipReasons = append(kubeFile.SkipReasons, fmt.Sprintf("owned by %+v", kubeFile.Parsed.Metadata.OwnerReferences))
		}
		if kubeFile.Parsed.Kind == "Endpoints" {
			kubeFile.SkipReasons = append(kubeFile.SkipReasons, "Endpoints are managed by K8S Services internally")
		}

		if kubeFile.Parsed.Kind == "EndpointSlice" {
			kubeFile.SkipReasons = append(kubeFile.SkipReasons, "EndpointSlice is managed by K8S Services internally")
		}

		if kubeFile.Parsed.Kind == "ServiceAccount" && kubeFile.Parsed.Metadata.Name == "default" {
			kubeFile.SkipReasons = append(kubeFile.SkipReasons, "default ServiceAccount is automatically created for each namespace")
		}

		if kubeFile.Parsed.Kind == "Secret" && some(kubeFiles, func(possibleServiceAccount *KubeFile) bool {
			// if the secret is referenced by a ServiceAccount...
			return possibleServiceAccount.Parsed.Kind == "ServiceAccount" &&
				len(possibleServiceAccount.Parsed.Secrets) > 0 &&
				possibleServiceAccount.Parsed.Secrets[0].Name == kubeFile.Parsed.Metadata.Name
		}) {
			// we skip the secret as it is autocreated when the serviceAccount is created.
			kubeFile.SkipReasons = append(kubeFile.SkipReasons, "Secret is auto-created by a ServiceAccount")
		}

		if kubeFile.Parsed.Kind == "Secret" && kubeFile.Parsed.Type == "helm.sh/release.v1" {
			kubeFile.SkipReasons = append(kubeFile.SkipReasons, "a helm secret")
		}

		if kubeFile.Parsed.Kind == "Secret" && kubeFile.Parsed.Type == "kubernetes.io/service-account-token" {
			kubeFile.SkipReasons = append(kubeFile.SkipReasons, "a service account token")
		}
	}

	return kubeFiles
}

func addExtraGlobalKubeFiles(kubeFiles []*KubeFile) []*KubeFile {
	for _, kubeFile := range kubeFiles {
		if kubeFile.Parsed.Kind == "ServiceAccount" && kubeFile.Parsed.Metadata.Name != "default" {
			// we have a non-default service account; let's check if there are global ClusterRoleBindings for this ServiceAccount
			globalFileList := buildFileListToRead("../../GLOBAL/config") // TODO
			globalKubeFiles := readKubeFiles(globalFileList)

			if globalClusterRoleBinding, isFound := findFirst(globalKubeFiles, func(globalKubeFile *KubeFile) bool {
				if globalKubeFile.Parsed.Kind != "ClusterRoleBinding" {
					// we're looking for global ClusterRoleBindings...
					return false
				}
				for _, subject := range globalKubeFile.Parsed.Subjects {
					// ... which have our ServiceAccount as subject
					if subject.Kind == "ServiceAccount" &&
						subject.Name == kubeFile.Parsed.Metadata.Name &&
						subject.Namespace == kubeFile.Parsed.Metadata.Namespace {
						// !! MATCH!
						return true
					}
				}
				return false
			}); isFound {
				// now, let's include the global ClusterRoleBinding in the output!
				kubeFiles = append(kubeFiles, globalClusterRoleBinding)

				// ... additionally, try to find the assigned ClusterRole
				if matchingClusterRole, isFound := findFirst(globalKubeFiles, func(globalKubeFile *KubeFile) bool {
					return globalKubeFile.Parsed.Kind == globalClusterRoleBinding.Parsed.RoleRef.Kind &&
						strings.Contains(globalKubeFile.Parsed.ApiVersion, globalClusterRoleBinding.Parsed.RoleRef.ApiGroup) &&
						globalKubeFile.Parsed.Metadata.Name == globalClusterRoleBinding.Parsed.RoleRef.Name
				}); isFound {
					// ... and add our ClusterRole (or Role) as well to the output
					kubeFiles = append(kubeFiles, matchingClusterRole)
				}
			}
		}
	}

	return kubeFiles
}

func some(kubeFiles []*KubeFile, callback func(*KubeFile) bool) bool {
	for _, kubeFile := range kubeFiles {
		if callback(kubeFile) {
			return true
		}
	}
	return false
}

func findFirst(kubeFiles []*KubeFile, callback func(*KubeFile) bool) (*KubeFile, bool) {
	for _, kubeFile := range kubeFiles {
		if callback(kubeFile) {
			return kubeFile, true
		}
	}
	return nil, false
}

func cleanManifests(kubeFiles []*KubeFile) []*KubeFile {
	for _, kubeFile := range kubeFiles {

		res, ok := kubeFile.FullKubeFile["metadata"]
		if !ok {
			// no metadata key
			continue
		}
		metadata, ok := res.(map[interface{}]interface{})
		if !ok {
			log.Fatalf("metadata was of type %T, expected map[interface{}]interface{}", res)
		}

		for k := range metadata {
			switch k {
			case "name", "namespace", "labels", "annotations":
			default:
				delete(metadata, k)
			}
		}

		metadata = cleanAnnotations(metadata)
		metadata = cleanLabels(metadata)

		kubeFile.FullKubeFile["metadata"] = metadata

		// Never restore status
		delete(kubeFile.FullKubeFile, "status")

	}

	return kubeFiles
}

func cleanAnnotations(metadata map[interface{}]interface{}) map[interface{}]interface{} {
	res, ok := metadata["annotations"]
	if !ok {
		// no annotations key
		return metadata
	}
	annotations, ok := res.(map[interface{}]interface{})
	if !ok {
		log.Fatalf("metadata[annotations] was of type %T, expected map[interface{}]interface{}", res)
	}

	for k := range annotations {
		if k == "kubectl.kubernetes.io/last-applied-configuration" ||
			k == "deployment.kubernetes.io/revision" {
			delete(annotations, k)
		}
	}

	metadata["annotations"] = annotations

	if len(annotations) == 0 {
		delete(metadata, "annotations")
	}

	return metadata
}

func cleanLabels(metadata map[interface{}]interface{}) map[interface{}]interface{} {
	res, ok := metadata["labels"]
	if !ok {
		// no labels key
		return metadata
	}
	labels, ok := res.(map[interface{}]interface{})
	if !ok {
		log.Fatalf("metadata[labels] was of type %T, expected map[interface{}]interface{}", res)
	}

	for k := range labels {
		if k == "cattle.io/creator" {
			delete(labels, k)
		}
	}

	metadata["labels"] = labels

	if len(labels) == 0 {
		delete(metadata, "labels")
	}

	return metadata
}

func cleanManifestsTypeSpecific(kubeFiles []*KubeFile) []*KubeFile {
	for _, kubeFile := range kubeFiles {
		if kubeFile.Parsed.Kind == "ServiceAccount" {
			// for serviceAccounts, delete the "secrets" key (they will be regenerated anyways
			delete(kubeFile.FullKubeFile, "secrets")
		}
	}
	return kubeFiles
}

func buildFileListToRead(filename string) []string {
	fileStats, err := os.Stat(filename)
	if err != nil {
		log.Fatalf("File %s not found: %s", filename, err)
	}

	filesToRead := make([]string, 0, 0)
	if !fileStats.Mode().IsDir() {
		filesToRead = append(filesToRead, filename)
	} else {
		directory, err := os.Open(filename)
		if err != nil {
			log.Fatalf("failed opening directory: %s", err)
		}
		defer directory.Close()

		list, _ := directory.Readdirnames(0) // 0 to read all files and folders
		for _, name := range list {
			if strings.HasSuffix(name, ".yaml") {
				filesToRead = append(filesToRead, filepath.Join(filename, name))
			}
		}
	}

	return filesToRead
}
