{
  // hidden k namespace for this library
  k:: {
    prometheusrule: {
      new(name, labels, rules): {
        apiVersion: 'monitoring.coreos.com/v1',
        kind: 'PrometheusRule',
        metadata: {
          labels: labels,
          name: name,
        },
        spec: rules,
      },
    },
  },
}
