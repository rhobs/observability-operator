package uiplugin

import (
	osv1 "github.com/openshift/api/console/v1"
)

type PluginProxy struct {
	Alias            string
	ServiceName      string
	ServiceNamespace string
	ServicePort      int32
	Authorize        bool
}

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
