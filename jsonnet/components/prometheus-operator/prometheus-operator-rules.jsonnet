local rules = import 'github.com/rhobs/obo-prometheus-operator/jsonnet/mixin/alerts.jsonnet';

{
  _commonLabels:: {
    'app.kubernetes.io/component': 'operator',
    'app.kubernetes.io/name': 'observability-operator-prometheus-operator-rules',
    'app.kubernetes.io/part-of': 'observability-operator',
    prometheus: 'k8s',
    role: 'alert-rules',
  },

  rule: $.k.prometheusrule.new('observability-operator-prometheus-operator-rules', $._commonLabels, rules),
}
