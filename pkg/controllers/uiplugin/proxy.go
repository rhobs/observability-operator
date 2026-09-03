package uiplugin

type PluginProxy struct {
	Alias            string
	ServiceName      string
	ServiceNamespace string
	ServicePort      int32
	Authorize        bool
}
