observability-operator must-gather
=================

`observability-operator-must-gather` is a tool built on top of [OpenShift must-gather](https://github.com/openshift/must-gather)
that expands its capabilities to gather Observability Operator information.

**Note:** This image is only built for x86_64 architecture

### Usage
To gather only Observability Operator information:
```sh
oc adm must-gather --image=quay.io/rhobs/observability-operator:latest -- /usr/bin/gather
```

To gather default [OpenShift must-gather](https://github.com/openshift/must-gather) in addition to Observability Operator information: 
```sh
oc adm must-gather --image-stream=openshift/must-gather --image=quay.io/rhobs/observability-operator -- /usr/bin/gather
```

The command above will create a local directory with a dump of the Observability Operator state.

You will get a dump of:
- The observability-operator operator deployment
- All observability-operator operant pods
- Alertmanager and Prometheus status for all stacks

In order to get data about other parts of the cluster (not specific to observability-operator ) you should
run `oc adm must-gather` (without passing a custom image). Run `oc adm must-gather -h` to see more options.

Example must-gather for observability-operator output:
```
monitoring
└── observability-operator
    ├── [namespace name]
    │   └── [monitoring stack name]
    │       ├── alertmanager
    │       │   ├── status.json
    │       │   └── status.stderr
    │       └── prometheus
    │           ├── alertmanagers.json
    │           ├── alertmanagers.stderr
    │           ├── prometheus-[monitoring stack name]-[replica]
    │           │   ├── status
    │           │   │   ├── runtimeinfo.json
    │           │   │   ├── runtimeinfo.stderr
    │           │   │   ├── tsdb.json
    │           │   │   └── tsdb.stderr
    │           │   ├── targets-active.json
    │           │   ├── targets-active.stderr
    │           │   ├── targets?state=active.json
    │           │   └── targets?state=active.stderr
    │           ├── rules.json
    │           ├── rules.stderr
    │           └── status
    │               ├── config.json
    │               ├── config.stderr
    │               ├── flags.json
    │               └── flags.stderr
    ├── operants.yaml
    └── operator.yaml
```
