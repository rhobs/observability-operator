package e2e

import (
	"context"
	"flag"
	"net/http"
	"sort"
	"strings"
	"testing"
	"time"

	routev1 "github.com/openshift/api/route/v1"
	"gotest.tools/v3/assert"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"

	uiv1 "github.com/rhobs/observability-operator/pkg/apis/uiplugin/v1alpha1"
	"github.com/rhobs/observability-operator/test/e2e/framework"
)

var uiPluginInstallNS string

func TestUIPlugin(t *testing.T) {
	if !f.IsOpenshiftCluster {
		t.Skip("The tests are skipped on non-ocp cluster")
	}

	flag.Parse()
	uiPluginInstallNS = *operatorInstallNS

	assertCRDExists(t, "uiplugins.observability.openshift.io")

	ts := []testCase{
		{
			name:     "Cluster health analyzer",
			scenario: clusterHealthAnalyzer,
		},
		{
			name:     "Create troubleshooting-panel UIPlugin",
			scenario: troubleshootingPanelUIPlugin,
		},
	}

	for _, tc := range ts {
		t.Run(tc.name, tc.scenario)
	}
}

func troubleshootingPanelUIPlugin(t *testing.T) {
	f.SkipIfClusterVersionBelow(t, "4.22")

	f.DumpOnFailure(t, f.DebugNamespaces(uiPluginInstallNS))

	tp := newTroubleshootingPanelUIPlugin(t)
	err := f.K8sClient.Create(t.Context(), tp)
	assert.NilError(t, err, "failed to create a troubleshooting-panel UIPlugin")

	tpDeployment := appsv1.Deployment{}
	f.GetResourceWithRetry(t, uiv1.TroubleshootingPanelPluginName, uiPluginInstallNS, &tpDeployment)
	f.AssertDeploymentReady(uiv1.TroubleshootingPanelPluginName, uiPluginInstallNS, framework.WithTimeout(5*time.Minute))(t)

	korrel8rDeployment := appsv1.Deployment{}
	f.GetResourceWithRetry(t, "korrel8r", uiPluginInstallNS, &korrel8rDeployment)
	f.AssertDeploymentReady("korrel8r", uiPluginInstallNS, framework.WithTimeout(5*time.Minute))(t)

	korrel8rRoute := routev1.Route{}
	f.GetResourceWithRetry(t, "korrel8r", uiPluginInstallNS, &korrel8rRoute)
	assert.Equal(t, korrel8rRoute.Spec.To.Name, "korrel8r")
	assert.Assert(t, korrel8rRoute.Spec.Port != nil, "korrel8r route has no target port")
	assert.Equal(t, korrel8rRoute.Spec.Port.TargetPort.StrVal, "web")
	assert.Assert(t, korrel8rRoute.Spec.TLS != nil, "korrel8r route has no TLS configuration")
	assert.Equal(t, korrel8rRoute.Spec.TLS.Termination, routev1.TLSTerminationReencrypt)

	// Use cluster credentials, but do not verify the ingress certificate (host may not trust it).
	insecureConfig := rest.CopyConfig(f.Config)
	insecureConfig.Insecure = true
	insecureConfig.CAFile = ""
	insecureConfig.CAData = nil
	// Collect authScheme and headerNames for debugging. No credential values.
	// WrapTransport runs inside client-go's auth wrapper, so it sees the headers actually sent.
	authScheme := "none"
	var headerNames []string
	previousWrap := insecureConfig.WrapTransport
	insecureConfig.WrapTransport = func(rt http.RoundTripper) http.RoundTripper {
		if previousWrap != nil {
			rt = previousWrap(rt)
		}
		return roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			headerNames = headerNames[:0]
			for name := range req.Header {
				headerNames = append(headerNames, name)
			}
			sort.Strings(headerNames)
			switch {
			case strings.HasPrefix(req.Header.Get("Authorization"), "Bearer "):
				authScheme = "Bearer"
			case strings.HasPrefix(req.Header.Get("Authorization"), "Basic "):
				authScheme = "Basic"
			case req.Header.Get("Authorization") != "":
				authScheme = "other"
			default:
				authScheme = "none"
			}
			return rt.RoundTrip(req)
		})
	}
	transport, err := rest.TransportFor(insecureConfig)
	assert.NilError(t, err)
	httpClient := &http.Client{Transport: transport, Timeout: 30 * time.Second}
	assert.Assert(t, korrel8rRoute.Spec.Host != "", "korrel8r route has no host")
	url := "https://" + korrel8rRoute.Spec.Host + "/api/v1alpha1/domains"
	err = wait.PollUntilContextTimeout(t.Context(), 5*time.Second, 2*time.Minute, true, func(ctx context.Context) (bool, error) {
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return false, err
		}
		response, err := httpClient.Do(request)
		if err != nil {
			t.Logf("korrel8r route request to %s failed: %v", url, err)
			return false, nil
		}
		defer response.Body.Close()
		if response.StatusCode != http.StatusOK {
			t.Logf("korrel8r route request to %s returned %s", url, response.Status)
		}
		return response.StatusCode == http.StatusOK, nil
	})
	// On failure dump authScheme and headerNames for debugging.
	if err != nil {
		t.Logf("korrel8r request header names=%v; Authorization scheme=%s (all values redacted)", headerNames, authScheme)
		t.Logf("kube config authentication: bearerToken=%t bearerTokenFile=%t clientCertificate=%t execProvider=%t authProvider=%t basicAuth=%t impersonation=%t; ingressTLSVerification=%t",
			f.Config.BearerToken != "", f.Config.BearerTokenFile != "", f.Config.CertFile != "" || len(f.Config.CertData) > 0,
			f.Config.ExecProvider != nil, f.Config.AuthProvider != nil, f.Config.Username != "", f.Config.Impersonate.UserName != "", !insecureConfig.Insecure)
	}
	assert.NilError(t, err, "korrel8r /domains route did not return HTTP 200")
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func newTroubleshootingPanelUIPlugin(t *testing.T) *uiv1.UIPlugin {
	plugin := &uiv1.UIPlugin{
		ObjectMeta: metav1.ObjectMeta{
			Name: uiv1.TroubleshootingPanelPluginName,
		},
		Spec: uiv1.UIPluginSpec{
			Type: uiv1.TypeTroubleshootingPanel,
		},
	}

	deleteUIPluginIfExists(t, plugin.Name)

	f.CleanUp(t, func() {
		ctx := context.WithoutCancel(t.Context())
		if err := f.K8sClient.Delete(ctx, plugin); err != nil && !errors.IsNotFound(err) {
			t.Logf("warning: failed to delete troubleshooting-panel UIPlugin during cleanup: %v", err)
		}
		waitForUIPluginDeletion(plugin)
	})
	return plugin
}

func waitForUIPluginDeletion(db *uiv1.UIPlugin) error {
	return wait.PollUntilContextTimeout(context.Background(), 5*time.Second, wait.ForeverTestTimeout, true, func(ctx context.Context) (done bool, err error) {
		err = f.K8sClient.Get(context.Background(),
			client.ObjectKey{Name: db.Name},
			db)
		return errors.IsNotFound(err), nil
	})
}
