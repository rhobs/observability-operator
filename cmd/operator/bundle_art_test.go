package main

import (
	"maps"
	"os"
	"slices"
	"strings"
	"testing"

	obopo "github.com/rhobs/obo-prometheus-operator/pkg/operator"
	"gotest.tools/v3/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apixv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	k8sflag "k8s.io/component-base/cli/flag"
	"sigs.k8s.io/yaml"
)

// TestARTBundleImages keeps the rendered disconnected bundle aligned with the
// operator's accepted flags, defaults, and the images ART must replace.
func TestARTBundleImages(t *testing.T) {
	var csv struct {
		Spec struct {
			Install struct {
				Spec struct {
					Deployments []struct {
						Name string                `json:"name"`
						Spec appsv1.DeploymentSpec `json:"spec"`
					} `json:"deployments"`
				} `json:"spec"`
			} `json:"install"`
			RelatedImages []struct {
				Image string `json:"image"`
			} `json:"relatedImages"`
		} `json:"spec"`
	}
	data, err := os.ReadFile("../../bundle-art/manifests/cluster-observability-operator.clusterserviceversion.yaml")
	assert.NilError(t, err)
	assert.NilError(t, yaml.Unmarshal(data, &csv))

	var imageReferences struct {
		Spec struct {
			Tags []struct {
				From struct {
					Name string `json:"name"`
				} `json:"from"`
			} `json:"tags"`
		} `json:"spec"`
	}
	data, err = os.ReadFile("../../bundle-art/image-references")
	assert.NilError(t, err)
	assert.NilError(t, yaml.Unmarshal(data, &imageReferences))

	references := map[string]bool{}
	for _, tag := range imageReferences.Spec.Tags {
		references[tag.From.Name] = true
	}
	related := map[string]bool{}
	for _, image := range csv.Spec.RelatedImages {
		related[image.Image] = true
	}

	used := map[string]bool{}
	var operatorContainer *corev1.Container
	for _, deployment := range csv.Spec.Install.Spec.Deployments {
		for _, container := range deployment.Spec.Template.Spec.Containers {
			used[container.Image] = true
			for _, env := range container.Env {
				if strings.HasPrefix(env.Name, "RELATED_IMAGE_") {
					used[env.Value] = true
				}
			}
			if deployment.Name == "observability-operator" && container.Name == "operator" {
				operatorContainer = &container
			}
		}
	}
	assert.Assert(t, operatorContainer != nil, "operator container not found")
	assert.DeepEqual(t, references, related)
	assert.DeepEqual(t, references, used)

	envs := map[string]string{}
	for _, env := range operatorContainer.Env {
		envs[env.Name] = env.Value
	}
	images := map[string]string{}
	imageFlags := k8sflag.NewMapStringString(new(map[string]string))
	for _, arg := range operatorContainer.Args {
		value, ok := strings.CutPrefix(arg, "--images=")
		if !ok {
			continue
		}
		key, envRef, ok := strings.Cut(value, "=")
		assert.Assert(t, ok, "invalid image argument: %s", arg)
		envName, ok := strings.CutPrefix(envRef, "$(")
		assert.Assert(t, ok, "image argument must reference an environment variable: %s", arg)
		envName, ok = strings.CutSuffix(envName, ")")
		assert.Assert(t, ok, "invalid environment variable reference: %s", arg)
		image, ok := envs[envName]
		assert.Assert(t, ok && image != "", "missing image environment variable %s", envName)
		_, duplicate := images[key]
		assert.Assert(t, !duplicate, "duplicate image argument: %s", key)
		images[key] = image
		assert.NilError(t, imageFlags.Set(key+"="+image))
	}

	expected := maps.Clone(defaultImages)
	expected["prometheus"] = obopo.DefaultPrometheusImage
	expected["alertmanager"] = obopo.DefaultAlertmanagerImage
	assert.DeepEqual(t, expected, images)

	// validateImages modifies its default map, so restore it after the test.
	savedDefaults := maps.Clone(defaultImages)
	t.Cleanup(func() { defaultImages = savedDefaults })
	_, err = validateImages(imageFlags)
	assert.NilError(t, err)
}

func TestPersesDatasourceConversionWebhook(t *testing.T) {
	for _, path := range []string{
		"../../deploy/perses/crds/patches/webhook_in_persesdatasource.yaml",
		"../../bundle/manifests/perses.dev_persesdatasources.yaml",
		"../../bundle-art/manifests/perses.dev_persesdatasources.yaml",
	} {
		t.Run(path, func(t *testing.T) {
			data, err := os.ReadFile(path)
			assert.NilError(t, err)
			var crd apixv1.CustomResourceDefinition
			assert.NilError(t, yaml.Unmarshal(data, &crd))
			assert.Equal(t, crd.Name, "persesdatasources.perses.dev")
			assert.Assert(t, crd.Spec.Conversion != nil)
			assert.Equal(t, crd.Spec.Conversion.Strategy, apixv1.WebhookConverter)
			webhook := crd.Spec.Conversion.Webhook
			assert.Assert(t, webhook != nil && webhook.ClientConfig != nil && webhook.ClientConfig.Service != nil)
			assert.DeepEqual(t, webhook.ConversionReviewVersions, []string{"v1"})
			service := webhook.ClientConfig.Service
			assert.Equal(t, service.Name, "perses-operator-conversion-webhook")
			assert.Equal(t, service.Namespace, "system")
			assert.Assert(t, service.Path != nil)
			assert.Equal(t, *service.Path, "/convert")
		})
	}

	for _, path := range []string{
		"../../bundle/manifests/observability-operator.clusterserviceversion.yaml",
		"../../bundle-art/manifests/cluster-observability-operator.clusterserviceversion.yaml",
	} {
		t.Run(path, func(t *testing.T) {
			data, err := os.ReadFile(path)
			assert.NilError(t, err)
			var csv struct {
				Spec struct {
					WebhookDefinitions []struct {
						Type           string   `json:"type"`
						DeploymentName string   `json:"deploymentName"`
						WebhookPath    string   `json:"webhookPath"`
						ConversionCRDs []string `json:"conversionCRDs"`
					} `json:"webhookdefinitions"`
				} `json:"spec"`
			}
			assert.NilError(t, yaml.Unmarshal(data, &csv))
			found := false
			for _, webhook := range csv.Spec.WebhookDefinitions {
				if webhook.Type == "ConversionWebhook" && slices.Contains(webhook.ConversionCRDs, "persesdatasources.perses.dev") {
					assert.Equal(t, webhook.DeploymentName, "perses-operator")
					assert.Equal(t, webhook.WebhookPath, "/convert")
					found = true
				}
			}
			assert.Assert(t, found, "PersesDatasource conversion webhook not registered in CSV")
		})
	}
}
