package generator

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"

	"github.com/go-logr/logr"
	configv1 "github.com/openshift/api/config/v1"
	"github.com/openshift/controller-runtime-common/pkg/tls"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/dynamic"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	k8sflag "k8s.io/component-base/cli/flag"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	kyaml "sigs.k8s.io/yaml"

	"github.com/rhobs/observability-operator/config"
	obsv1alpha1 "github.com/rhobs/observability-operator/pkg/apis/observability/v1alpha1"
	uiv1alpha1 "github.com/rhobs/observability-operator/pkg/apis/uiplugin/v1alpha1"
	"github.com/rhobs/observability-operator/pkg/controllers/observability"
	"github.com/rhobs/observability-operator/pkg/controllers/uiplugin"
	"github.com/rhobs/observability-operator/pkg/controllers/util"
	"github.com/rhobs/observability-operator/pkg/images"
	"github.com/rhobs/observability-operator/pkg/operator"
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

// RunConfig holds the inputs for a generator run. It is populated by the CLI
// flags in Main and directly by tests.
type RunConfig struct {
	// Input is the list of YAML files, directories, or "-" for stdin.
	Input []string
	// Namespace is the namespace the operator itself is installed in.
	Namespace string
	// ClusterVersion is the OpenShift cluster version, used by UIPlugin code.
	ClusterVersion string
	// Operators selects operator infrastructure only; Resources selects
	// non-operator resources only. Both false or both true means everything.
	Operators bool
	Resources bool
	// OtelCSV and TempoCSV are the starting CSVs for the OTel and Tempo
	// subscriptions (empty means the operator catalog's default).
	OtelCSV  string
	TempoCSV string
	// Images overrides default images, keyed by image name.
	Images map[string]string
	// K8sClient, when set, is used to fetch version, TLS profile and secrets.
	K8sClient     client.Client
	DynamicClient dynamic.Interface
	// TLSProfile overrides the TLS profile used for UIPlugin operands.
	TLSProfile configv1.TLSProfileSpec
}

// Run generates the YAML manifest for the configured resources.
// It returns the stdout manifest, and errors collected during the run.
func Run(cfg RunConfig) (string, string, error) {
	resolvedImages, err := images.Validate(cfg.Images)
	if err != nil {
		return "", "", err
	}
	// Default values from cfg
	clusterVersion := cfg.ClusterVersion
	tlsProfile := cfg.TLSProfile
	if cfg.K8sClient != nil {
		// Overwrite with cluster values if we have a K8sClient
		tlsProfile, err = tls.FetchAPIServerTLSProfile(context.Background(), cfg.K8sClient)
		if err != nil {
			return "", "", fmt.Errorf("error fetching TLS profile from cluster: %w", err)
		}
		cv := &configv1.ClusterVersion{}
		err = cfg.K8sClient.Get(context.Background(), client.ObjectKey{Name: "version"}, cv)
		if err != nil {
			return "", "", fmt.Errorf("error fetching cluster version: %w", err)
		}
		for _, update := range cv.Status.History {
			if update.State == configv1.CompletedUpdate {
				clusterVersion = update.Version
				break
			}
		}
	}

	scheme := operator.NewScheme(&operator.OperatorConfiguration{
		FeatureGates: operator.FeatureGates{
			OpenShift: operator.OpenShiftFeatureGates{
				Enabled: true,
				Version: clusterVersion,
			},
		}})

	yamlData, err := readInputs(cfg.Input)
	if err != nil {
		return "", "", fmt.Errorf("error reading input: %w", err)
	}

	installers, plugins, others, err := decodeResources(scheme, yamlData)
	if err != nil {
		return "", "", fmt.Errorf("error decoding resources: %w", err)
	}

	installerOpts := observability.Options{
		COONamespace: cfg.Namespace,
		OpenTelemetryOperator: observability.OperatorInstallConfig{
			Namespace:   cfg.Namespace,
			PackageName: "opentelemetry-product",
			StartingCSV: cfg.OtelCSV,
			Channel:     "stable",
		},
		TempoOperator: observability.OperatorInstallConfig{
			Namespace:   cfg.Namespace,
			PackageName: "tempo-product",
			StartingCSV: cfg.TempoCSV,
			Channel:     "stable",
		},
	}

	installOperators := cfg.Operators || !cfg.Resources
	installResources := cfg.Resources || !cfg.Operators

	// accumulate resources in set
	set := newResourceSet(scheme)

	// add the COO namespace to the set, for the operator this is created by OLM
	if err := set.AddResource(&corev1.Namespace{
		TypeMeta:   metav1.TypeMeta{Kind: "Namespace", APIVersion: corev1.SchemeGroupVersion.String()},
		ObjectMeta: metav1.ObjectMeta{Name: cfg.Namespace},
	}); err != nil {
		return "", "", fmt.Errorf("adding COO namespace: %w", err)
	}

	// pass through other resources to the output
	for _, obj := range others {
		if err := set.AddResource(obj); err != nil {
			return "", "", fmt.Errorf("adding input resource: %w", err)
		}
	}

	// reader makes other objects (secrets, configmaps) read from YAML available to kustomize.
	reader := NewFallbackReader(cfg.K8sClient, others...)

	var errs error
	err = generateInstallerObjects(installers, installerOpts, set, reader, installOperators, installResources)
	errs = errors.Join(errs, err)

	if installResources {
		tracingInstaller := findTracingInstaller(installers)
		pluginInfo := uiplugin.UIPluginInfo{
			Scheme:            scheme,
			ConfigFS:          config.FS,
			Images:            resolvedImages,
			Namespace:         cfg.Namespace,
			ClusterVersion:    clusterVersion,
			TracingInstaller:  tracingInstaller,
			TempoServiceNames: tempoServiceNamesFromInstaller(tracingInstaller),
		}
		pluginInfo.ApplyTLSProfile(tlsProfile)

		if err := resolveUIPlugins(plugins, pluginInfo, set); err != nil {
			errs = errors.Join(errs, fmt.Errorf("resolving UIPlugins: %w", err))
		}
	}

	objects := set.Build()

	// UIPlugin CRs have been resolved into concrete resources, remove them from output.
	objects = slices.DeleteFunc(objects, func(obj client.Object) bool {
		gvk := obj.GetObjectKind().GroupVersionKind()
		return gvk.Group == "observability.openshift.io" && gvk.Kind == "UIPlugin"
	})

	var buf bytes.Buffer
	for _, obj := range objects {
		data, err := kyaml.Marshal(obj)
		if err != nil {
			return "", "", err
		}
		buf.WriteString("---\n")
		buf.Write(data)
	}

	return buf.String(), errsString(errs), nil
}

func errsString(errs error) string {
	if errs == nil {
		return ""
	}
	return errs.Error()
}

func Main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `
Usage: %v [OPTIONS] -f FILE...

Offline generator for the Cluster Observability Operator (COO). Reads COO custom
resources and supporting objects from YAML, runs the operator's reconciler code
offline, and writes the resulting Kubernetes objects to stdout. Use -operators
and -resources for staged apply; use -cluster to resolve secrets, TLS profile,
and version from a live cluster.
`, filepath.Base(os.Args[0]))
		flag.PrintDefaults()
	}
	var files []string
	filesFlag := k8sflag.NewStringSlice(&files)
	flag.Var(filesFlag, "f", "YAML file, directory, or - for stdin (repeatable)")
	clusterVersion := flag.String("cluster-version", "v4.22", "OpenShift cluster version (overridden by -cluster)")
	namespace := flag.String("namespace", "openshift-cluster-observability-operator", "Namespace for operator resources")
	useCluster := flag.Bool("cluster", false, "Connect to cluster for version, secrets, and TLS profile")
	otelCSV := flag.String("opentelemetry-csv", "", "OpenTelemetry Operator starting CSV (default: latest)")
	tempoCSV := flag.String("tempo-csv", "", "Tempo Operator starting CSV (default: latest)")
	operators := flag.Bool("operators", false, "Output only operator resources (Subscriptions)")
	resources := flag.Bool("resources", false, "Output only non-operator resources (CRD instances, RBAC, etc.)")
	imageOverrides := k8sflag.NewMapStringString(new(map[string]string))
	flag.Var(imageOverrides, "images", "Override default image as key=value (repeatable)")
	flag.Parse()
	if flag.NArg() > 0 {
		exitMsg("unused arguments: %v", flag.Args())
	}

	if len(files) == 0 {
		exitMsg("no input specified, use -f")
	}

	var k8sClient client.Client
	var dynamicClient dynamic.Interface
	if *useCluster {
		restConfig := ctrl.GetConfigOrDie()
		clusterScheme := runtime.NewScheme()
		exitErr(clientgoscheme.AddToScheme(clusterScheme), "error loading scheme")
		exitErr(configv1.Install(clusterScheme), "error loading OpenShift config scheme")

		var err error
		k8sClient, err = client.New(restConfig, client.Options{Scheme: clusterScheme})
		exitErr(err, "error creating cluster client")
		dynamicClient, err = dynamic.NewForConfig(restConfig)
		exitErr(err, "error creating dynamic client")
	}

	stdout, warnings, err := Run(RunConfig{
		Input:          files,
		Namespace:      *namespace,
		ClusterVersion: *clusterVersion,
		Operators:      *operators,
		Resources:      *resources,
		OtelCSV:        *otelCSV,
		TempoCSV:       *tempoCSV,
		Images:         *imageOverrides.Map,
		K8sClient:      k8sClient,
		DynamicClient:  dynamicClient,
	})
	exitErr(err, "error generating resources")

	if stdout != "" {
		fmt.Print(stdout)
	}
	if warnings != "" {
		fmt.Fprintln(os.Stderr, warnings)
		os.Exit(1)
	}
}

// generateInstallerObjects generates the operands of each installer and adds to set.
func generateInstallerObjects(installers []*obsv1alpha1.ObservabilityInstaller, opts observability.Options, set *resourceSet, reader client.Reader, installOperators, installResources bool) error {
	var errs error

	for _, installer := range installers {
		if installer.Namespace == "" {
			errs = errors.Join(errs, fmt.Errorf("skipping ObservabilityInstaller %q: no namespace set (use metadata.namespace or kubectl -n)", installer.Name))
			continue
		}
		installerObjects, err := observability.GenerateInstallerObjects(context.Background(), reader, installer, opts, installOperators, installResources)
		if err != nil {
			errs = errors.Join(errs, err)
			continue
		}
		for _, obj := range installerObjects {
			if err := set.AddResource(obj); err != nil {
				errs = errors.Join(errs, err)
			}
		}
	}
	return errs
}

func findTracingInstaller(installers []*obsv1alpha1.ObservabilityInstaller) *obsv1alpha1.ObservabilityInstaller {
	for _, installer := range installers {
		if tracing := installer.Spec.GetCapabilities().GetTracing(); tracing != nil && tracing.Enabled {
			return installer
		}
	}
	return nil
}

func tempoServiceNamesFromInstaller(installer *obsv1alpha1.ObservabilityInstaller) map[string]string {
	if installer == nil {
		return nil
	}
	return map[string]string{
		installer.Namespace: fmt.Sprintf("tempo-%s-gateway", installer.Name),
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

// resolveUIPlugins generates the operands of each uiplugin and adds to the set.
func resolveUIPlugins(plugins []*uiv1alpha1.UIPlugin, conf uiplugin.UIPluginInfo, set *resourceSet) error {
	// Add UIPlugin CRs generated by the observabilityinstaller.
	uiPluginGVK := uiv1alpha1.GroupVersion.WithKind("UIPlugin")
	for _, obj := range set.Build() {
		if obj.GetObjectKind().GroupVersionKind() != uiPluginGVK {
			continue
		}
		plugin := &uiv1alpha1.UIPlugin{}
		// MarshalCopy works for structured or unstructured obj
		if err := util.MarshalCopy(plugin, obj); err != nil {
			return fmt.Errorf("extracting UIPlugin %s: %w", obj.GetName(), err)
		}
		plugins = append(plugins, plugin)
	}
	slices.SortFunc(plugins, func(a, b *uiv1alpha1.UIPlugin) int { return util.CompareObjects(a, b) })
	plugins = slices.CompactFunc(plugins, func(a, b *uiv1alpha1.UIPlugin) bool { return util.CompareObjects(a, b) == 0 })

	var errs error
	for _, plugin := range plugins {
		pluginObjects, _, err := uiplugin.ResolvePlugin(plugin, conf, logr.Discard())
		errs = errors.Join(errs, err)
		for _, obj := range pluginObjects {
			util.AddCommonLabels(obj, plugin.Name)
			if err := set.AddResource(obj); err != nil {
				errs = errors.Join(errs, err)
			}
		}
	}
	return errs
}

// decodeResources returns installers, uiplugins and other objects like secrets and configmaps.
func decodeResources(scheme *runtime.Scheme, data []byte) ([]*obsv1alpha1.ObservabilityInstaller, []*uiv1alpha1.UIPlugin, []client.Object, error) {
	decode := serializer.NewCodecFactory(scheme).UniversalDeserializer().Decode
	reader := yaml.NewYAMLReader(bufio.NewReader(bytes.NewReader(data)))

	var installers []*obsv1alpha1.ObservabilityInstaller
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
			// Fall back to unstructured for types not in the scheme.
			var u unstructured.Unstructured
			if jsonErr := kyaml.Unmarshal(doc, &u.Object); jsonErr != nil {
				return nil, nil, nil, fmt.Errorf("decoding YAML document: %w", err)
			}
			if u.GetKind() != "" {
				loadedObjects = append(loadedObjects, &u)
			}
			continue
		}

		switch o := obj.(type) {
		case *obsv1alpha1.ObservabilityInstaller:
			installers = append(installers, o)
		case *uiv1alpha1.UIPlugin:
			plugins = append(plugins, o)
		default:
			if co, ok := o.(client.Object); ok {
				loadedObjects = append(loadedObjects, co)
			}
		}
	}
	return installers, plugins, loadedObjects, nil
}
