package assets

import (
	"os"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"

	"k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Asset struct {
	File   string
	Object client.Object
}

// NewCRDAsset returns a new Asset with type CustomResourceDefinition
func NewCRDAsset(file string) Asset {
	return Asset{
		File:   file,
		Object: &apiextensionsv1.CustomResourceDefinition{},
	}
}

// Loader loads Kubernetes objects from YAML manifests
type Loader struct {
	assetsPath string
}

// NewLoader returns a new Loader for assets in assetsPath
func NewLoader(assetsPath string) *Loader {
	return &Loader{
		assetsPath: assetsPath,
	}
}

// Load parses YAML manifests from disk and returns
// the corresponding resources as golang objects
func (l *Loader) Load(assets []Asset) ([]client.Object, error) {
	resources := make([]client.Object, len(assets))
	for i, asset := range assets {
		file, err := os.Open(l.assetsPath + asset.File)
		if err != nil {
			return nil, err
		}
		decoder := yaml.NewYAMLOrJSONDecoder(file, 0)
		if err := decoder.Decode(asset.Object); err != nil {
			return nil, err
		}

		resources[i] = asset.Object
	}

	return resources, nil
}
