package generator

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-logr/logr"
	configv1 "github.com/openshift/api/config/v1"
	openshifttls "github.com/openshift/controller-runtime-common/pkg/tls"
	libgocrypto "github.com/openshift/library-go/pkg/crypto"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/util/yaml"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/kustomize/api/krusty"
	"sigs.k8s.io/kustomize/kyaml/filesys"
	kyaml "sigs.k8s.io/yaml"

	"github.com/rhobs/observability-operator/config"
	obsv1alpha1 "github.com/rhobs/observability-operator/pkg/apis/observability/v1alpha1"
	uiv1alpha1 "github.com/rhobs/observability-operator/pkg/apis/uiplugin/v1alpha1"
	"github.com/rhobs/observability-operator/pkg/controllers/observability"
	"github.com/rhobs/observability-operator/pkg/controllers/uiplugin"
	"github.com/rhobs/observability-operator/pkg/images"
	"github.com/rhobs/observability-operator/pkg/overlay"
)

func exitMsg(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

func exitErr(err error, msg string) {
	if err != nil {
		exitMsg("%v: %v", msg, err)
	}
}

// stringsFlag implements flag.Value
type stringsFlag []string

func (s *stringsFlag) String() string { return fmt.Sprintf("%v", *s) }
func (s *stringsFlag) Set(val string) error {
	*s = append(*s, val)
	return nil
}

// mapFlag implements flag.Value for key=value pairs.
type mapFlag map[string]string

func (m *mapFlag) String() string { return fmt.Sprintf("%v", *m) }
func (m *mapFlag) Set(val string) error {
	k, v, ok := strings.Cut(val, "=")
	if !ok {
		return fmt.Errorf("expected key=value, got %q", val)
	}
	(*m)[k] = v
	return nil
}

func Main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `
Usage: %v [OPTIONS] (-f FILE... | -k DIR)

Reconcile Cluster Observability Operator custom resources and write as YAML.
The input resources should be COO custom resources such as ObservabilityInstaller or UIPLugin,
or supporting resources such as Secret or ConfigMap.

`, filepath.Base(os.Args[0]))
		flag.PrintDefaults()
	}
	var files stringsFlag
	flag.Var(&files, "f", "YAML file, directory, or - for stdin (can be repeated)")
	kustomizeInDir := flag.String("k", "", "Run kustomize build on DIR and use the output as input (like kubectl apply -k)")
	clusterVersion := flag.String("cluster-version", "4.22", "OpenShift cluster version for compatibility matrix")
	namespace := flag.String("namespace", "openshift-cluster-observability-operator", "Default namespace for resources")
	configPath := flag.String("config", "", "Path to input config directory (default: use embedded config)")
	saveDir := flag.String("save", "", "Directory to save intermediate kustomization, overlay SAVE/overlays/generated")
	outputDir := flag.String("output", "", "Directory for output resources, instead of stdout")
	useCluster := flag.Bool("cluster", false, "Fetch missing resources (secrets, TLS profile) from the cluster")
	otelCSV := flag.String("opentelemetry-csv", "", "OpenTelemetry Operator starting CSV name. Empty string means the latest version will be installed.")
	tempoCSV := flag.String("tempo-csv", "", "Tempo Operator starting CSV name. Empty string means the latest version will be installed.")
	imageOverrides := mapFlag{}
	flag.Var(&imageOverrides, "images", "Override default image, e.g. korrel8r=quay.io/korrel8r/korrel8r@sha256:abc. Can be repeated.")
	flag.Parse()
	if flag.NArg() > 0 {
		exitMsg(fmt.Sprintf("unused arguments: %v", flag.Args()))
	}

	scheme := runtime.NewScheme()
	exitErr(clientgoscheme.AddToScheme(scheme), "error loading scheme")
	exitErr(obsv1alpha1.AddToScheme(scheme), "error loading scheme")
	exitErr(uiv1alpha1.AddToScheme(scheme), "error loading scheme")

	if len(files) == 0 && *kustomizeInDir == "" {
		exitMsg("no input specified, use -f or -k")
	}
	if len(files) > 0 && *kustomizeInDir != "" {
		exitMsg("-f and -k are mutually exclusive")
	}

	var (
		yamlData []byte
		err      error
	)
	if *kustomizeInDir != "" {
		yamlData, err = kustomizeBuild(*kustomizeInDir)
		exitErr(err, "error running kustomize build")
	} else {
		yamlData, err = readInputs(files)
		exitErr(err, "error reading input")
	}

	installer, plugins, others, err := decodeResources(scheme, yamlData)
	exitErr(err, "error decoding resources")

	resolvedImages := make(map[string]string, len(images.DefaultImages))
	for k, v := range images.DefaultImages {
		resolvedImages[k] = v
	}
	for k, v := range imageOverrides {
		if _, ok := resolvedImages[k]; !ok {
			exitMsg("unknown image key %q", k)
		}
		resolvedImages[k] = v
	}

	var configFS fs.FS = config.FS
	if *configPath != "" {
		configFS = os.DirFS(*configPath)
	}

	var k8sClient client.Client
	var tlsProfile configv1.TLSProfileSpec
	if *useCluster {
		clusterScheme := runtime.NewScheme()
		exitErr(clientgoscheme.AddToScheme(clusterScheme), "error loading scheme")
		exitErr(configv1.Install(clusterScheme), "error loading OpenShift config scheme")

		var err error
		k8sClient, err = client.New(ctrl.GetConfigOrDie(), client.Options{Scheme: clusterScheme})
		exitErr(err, "error creating cluster client")

		tlsProfile, err = openshifttls.FetchAPIServerTLSProfile(context.Background(), k8sClient)
		exitErr(err, "error fetching TLS profile from cluster")
	}

	cfg := observability.OverlayConfig{
		ConfigFS:     configFS,
		COOName:      "observability-operator",
		COONamespace: *namespace,
		OpenTelemetryOperator: observability.OperatorInstallConfig{
			StartingCSV: *otelCSV,
		},
		TempoOperator: observability.OperatorInstallConfig{
			StartingCSV: *tempoCSV,
		},
	}

	overlay, err := observability.BuildOverlay(installer, cfg)
	exitErr(err, "error building overlay")

	pluginConf := uiplugin.UIPluginBuildConfig{
		ConfigFS:       configFS,
		Images:         resolvedImages,
		OperatorName:   "observability-operator",
		Namespace:      *namespace,
		ClusterVersion: *clusterVersion,
		TLSMinVersion:  string(tlsProfile.MinTLSVersion),
		TLSCiphers:     libgocrypto.OpenSSLToIANACipherSuites(tlsProfile.Ciphers),
	}
	exitErr(resolveUIPlugins(overlay, plugins, pluginConf), "error resolving UIPlugins")

	// Resolve secrets/configmaps: check loaded input files first, then cluster.
	if tracing := installer.Spec.GetCapabilities().GetTracing(); tracing != nil && tracing.Enabled {
		secrets, err := observability.BuildTempoSecrets(context.Background(), NewFallbackReader(k8sClient, others...), *installer)
		exitErr(err, "error building secrets")
		for _, obj := range secrets {
			gvk := obj.GetObjectKind().GroupVersionKind()
			data, err := kyaml.Marshal(obj)
			exitErr(err, "error marshaling secret")
			name := fmt.Sprintf("%s-%s.yaml", strings.ToLower(gvk.Kind), obj.GetName())
			overlay.AddResource(name, data)
		}
	}

	if *saveDir != "" {
		err = os.MkdirAll(*saveDir, 0o755)
		exitErr(err, "error creating save directory")
		err = overlay.WriteToDir(*saveDir)
		exitErr(err, "error saving intermediate kustomization")
	}

	objects, err := overlay.Build()
	exitErr(err, "error building overlay")

	if *outputDir != "" && *outputDir != "-" {
		err = os.MkdirAll(*outputDir, 0o755)
		exitErr(err, "error creating output directory")
		for _, obj := range objects {
			data, err := kyaml.Marshal(obj)
			exitErr(err, "error marshaling resource")
			gvk := obj.GetObjectKind().GroupVersionKind()
			name := fmt.Sprintf("%s-%s.yaml", strings.ToLower(gvk.Kind), obj.GetName())
			err = os.WriteFile(filepath.Join(*outputDir, name), data, 0o644)
			exitErr(err, "error writing file")
		}
	} else {
		yamlOut, err := overlay.BuildYAML()
		exitErr(err, "error building overlay")
		_, err = os.Stdout.Write(yamlOut)
		exitErr(err, "error writing output")
	}
}

func readInputs(files []string) ([]byte, error) {
	var buf bytes.Buffer
	appendFile := func(path string) error {
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if buf.Len() > 0 {
			buf.WriteString("\n---\n")
		}
		buf.Write(data)
		return nil
	}
	for _, f := range files {
		if f == "-" {
			data, err := io.ReadAll(os.Stdin)
			if err != nil {
				return nil, err
			}
			if buf.Len() > 0 {
				buf.WriteString("\n---\n")
			}
			buf.Write(data)
			continue
		}
		info, err := os.Stat(f)
		if err != nil {
			return nil, err
		}
		if !info.IsDir() {
			if err := appendFile(f); err != nil {
				return nil, err
			}
			continue
		}
		entries, err := os.ReadDir(f)
		if err != nil {
			return nil, err
		}
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			ext := filepath.Ext(e.Name())
			if ext != ".yaml" && ext != ".yml" {
				continue
			}
			if err := appendFile(filepath.Join(f, e.Name())); err != nil {
				return nil, err
			}
		}
	}
	return buf.Bytes(), nil
}

func kustomizeBuild(dir string) ([]byte, error) {
	k := krusty.MakeKustomizer(krusty.MakeDefaultOptions())
	resMap, err := k.Run(filesys.MakeFsOnDisk(), dir)
	if err != nil {
		return nil, fmt.Errorf("kustomize build %s: %w", dir, err)
	}
	return resMap.AsYaml()
}

func resolveUIPlugins(o *overlay.Overlay, inputPlugins []*uiv1alpha1.UIPlugin, conf uiplugin.UIPluginBuildConfig) error {
	objects, err := o.Build()
	if err != nil {
		return fmt.Errorf("building overlay: %w", err)
	}

	var generatedPlugins []*uiv1alpha1.UIPlugin
	for _, obj := range objects {
		gvk := obj.GetObjectKind().GroupVersionKind()
		if gvk.Group == "observability.openshift.io" && gvk.Kind == "UIPlugin" {
			plugin := &uiv1alpha1.UIPlugin{}
			u := obj.(*unstructured.Unstructured)
			if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, plugin); err != nil {
				return fmt.Errorf("converting UIPlugin: %w", err)
			}
			generatedPlugins = append(generatedPlugins, plugin)
		}
	}

	generatedPlugins = append(generatedPlugins, inputPlugins...)
	if len(generatedPlugins) == 0 {
		return nil
	}

	for _, plugin := range generatedPlugins {
		pluginOverlay, _, err := uiplugin.BuildUIPluginOverlay(plugin, conf, logr.Discard())
		if err != nil {
			return fmt.Errorf("building UIPlugin %s: %w", plugin.Name, err)
		}

		pluginObjects, err := pluginOverlay.Build()
		if err != nil {
			return fmt.Errorf("building UIPlugin %s overlay: %w", plugin.Name, err)
		}

		for _, obj := range pluginObjects {
			data, err := kyaml.Marshal(obj)
			if err != nil {
				return fmt.Errorf("marshaling UIPlugin resource: %w", err)
			}
			gvk := obj.GetObjectKind().GroupVersionKind()
			name := fmt.Sprintf("uiplugin-%s-%s-%s.yaml", plugin.Name, gvk.Kind, obj.GetName())
			o.AddResource(name, data)
		}
	}
	return nil
}

// decodeResources returns installers, uiplugins and other objects like secrets and configmaps.
func decodeResources(scheme *runtime.Scheme, data []byte) (*obsv1alpha1.ObservabilityInstaller, []*uiv1alpha1.UIPlugin, []client.Object, error) {
	decode := serializer.NewCodecFactory(scheme).UniversalDeserializer().Decode
	reader := yaml.NewYAMLReader(bufio.NewReader(bytes.NewReader(data)))

	var installer *obsv1alpha1.ObservabilityInstaller
	var plugins []*uiv1alpha1.UIPlugin
	var loadedObjects []client.Object

	for {
		doc, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, nil, nil, fmt.Errorf("reading YAML document: %w", err)
		}
		doc = bytes.TrimSpace(doc)
		if len(doc) == 0 {
			continue
		}

		obj, _, err := decode(doc, nil, nil)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("decoding YAML document: %w", err)
		}

		switch o := obj.(type) {
		case *obsv1alpha1.ObservabilityInstaller:
			if installer != nil {
				return nil, nil, nil, fmt.Errorf("multiple ObservabilityInstaller resources found")
			}
			installer = o
		case *uiv1alpha1.UIPlugin:
			plugins = append(plugins, o)
		default:
			if co, ok := o.(client.Object); ok {
				loadedObjects = append(loadedObjects, co)
			}
		}
	}
	if installer == nil {
		installer = &obsv1alpha1.ObservabilityInstaller{}
	}
	return installer, plugins, loadedObjects, nil
}
