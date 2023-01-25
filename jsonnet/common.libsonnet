{
  // hidden k namespace for this library
  k:: {
    prometheusrule: {
      new(name, labels, rules): {
        apiVersion: 'monitoring.rhobs/v1',
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
