local rules = import 'github.com/prometheus-operator/prometheus-operator/jsonnet/mixin/alerts.jsonnet';

{
  _commonLabels:: {
    'app.kubernetes.io/component': 'operator',
    'app.kubernetes.io/name': 'monitoring-stack-operator-prometheus-operator-rules',
    'app.kubernetes.io/part-of': 'monitoring-stack-operator',
    prometheus: 'k8s',
    role: 'alert-rules',
  },

  rule: $.k.prometheusrule.new('monitoring-stack-operator-prometheus-operator-rules', $._commonLabels, rules),
}
