package uiplugin

import (
	osv1 "github.com/openshift/api/console/v1"
	osRhobsv1 "github.com/rhobs/openshift-api/console/v1"
	osv1alpha1 "github.com/rhobs/openshift-api/console/v1alpha1"
)

type PluginProxy struct {
	Alias            string
	ServiceName      string
	ServiceNamespace string
	ServicePort      int32
	Authorize        bool
}

// Used for OCP 4.16 and earlier.
func (p PluginProxy) ToV1Alpha1() osv1alpha1.ConsolePluginProxy {
	return osv1alpha1.ConsolePluginProxy{
		Type:      "Service",
		Alias:     p.Alias,
		Authorize: p.Authorize,
		Service: osv1alpha1.ConsolePluginProxyServiceConfig{
			Name:      p.ServiceName,
			Namespace: p.ServiceNamespace,
			Port:      p.ServicePort,
		},
	}
}

// Used for OCP 4.17-18 (No CSP Field)
func (p PluginProxy) ToRhobsV1() osRhobsv1.ConsolePluginProxy {
	authorization := osRhobsv1.None
	if p.Authorize {
		authorization = osRhobsv1.UserToken
	}

	return osRhobsv1.ConsolePluginProxy{
		Alias:         p.Alias,
		Authorization: authorization,
		Endpoint: osRhobsv1.ConsolePluginProxyEndpoint{
			Type: osRhobsv1.ProxyTypeService,
			Service: &osRhobsv1.ConsolePluginProxyServiceConfig{
				Name:      p.ServiceName,
				Namespace: p.ServiceNamespace,
				Port:      p.ServicePort,
			},
		},
	}
}

// Used for OCP 4.19+
func (p PluginProxy) ToUpstreamV1() osv1.ConsolePluginProxy {
	authorization := osv1.None
	if p.Authorize {
		authorization = osv1.UserToken
	}

	return osv1.ConsolePluginProxy{
		Alias:         p.Alias,
		Authorization: authorization,
		Endpoint: osv1.ConsolePluginProxyEndpoint{
			Type: osv1.ProxyTypeService,
			Service: &osv1.ConsolePluginProxyServiceConfig{
				Name:      p.ServiceName,
				Namespace: p.ServiceNamespace,
				Port:      p.ServicePort,
			},
		},
	}
}
