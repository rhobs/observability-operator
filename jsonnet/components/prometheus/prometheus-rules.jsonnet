/*Cannot use https://github.com/prometheus/prometheus/blob/main/documentation/prometheus-mixin/mixin.libsonnet since it generates yaml with double quotes wrapped*/

local rules = (
  import 'github.com/prometheus/prometheus/documentation/prometheus-mixin/mixin.libsonnet'
).prometheusAlerts;

{
  _commonLabels:: {
    'app.kubernetes.io/component': 'operator',
    'app.kubernetes.io/name': 'monitoring-stack-operator-prometheus-rules',
    'app.kubernetes.io/part-of': 'monitoring-stack-operator',
    prometheus: 'k8s',
    role: 'alert-rules',
  },

  rule: $.k.prometheusrule.new('monitoring-stack-operator-prometheus-rules', $._commonLabels, rules),
}
