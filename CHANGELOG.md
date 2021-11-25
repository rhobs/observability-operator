# Changelog

All notable changes to this project will be documented in this file. See [standard-version](https://github.com/conventional-changelog/standard-version) for commit guidelines.

### [0.0.4](https://github.com/rhobs/monitoring-stack-operator/commit/) (2021-11-25)


### Features

* deploy an Alertmanager instance for each monitoring stack ([e607afe](https://github.com/rhobs/monitoring-stack-operator/commit/e607afe23dd604845fad170d06a0cabb6aa1ad28))


### Bug Fixes

* ensure operator has no reconciliation errors ([5257706](https://github.com/rhobs/monitoring-stack-operator/commit/5257706d573c7e96adfd91b9d3e6565b168ab110))
* query Prometheus through a dedicated service  ([58586e8](https://github.com/rhobs/monitoring-stack-operator/commit/58586e8c7cdfb077713077aa08149a9745b22d5f))

### [0.0.3](https://github.com/rhobs/monitoring-stack-operator/commit/) (2021-11-10)


### Features


* add thanos querier CRD ([#52](https://github.com/rhobs/monitoring-stack-operator/issues/52)) ([0dd9499](https://github.com/rhobs/monitoring-stack-operator/commit/0dd94995b006c4df8b13326ae8ab8a9831eb23fc))
* deploy an instance of the grafana operator ([409a95e](https://github.com/rhobs/monitoring-stack-operator/commit/409a95e986b8f2a3151e327c20c6c2ae5c83b863))
* implement self-scraping for monitoring stacks ([632f913](https://github.com/rhobs/monitoring-stack-operator/commit/632f9133ae333bae49cbc33912c6d9093d533a24))
* monitoring-stack controller that deploys prometheus ([#40](https://github.com/rhobs/monitoring-stack-operator/issues/40)) ([f16a977](https://github.com/rhobs/monitoring-stack-operator/commit/f16a9772add878df90b37fc7cf2bd95f26ce94f3))
* deploy a default grafana instance ([b1455bd](https://github.com/rhobs/monitoring-stack-operator/commit/b1455bd3df5c5939383e0265f99b62d554b0df03))

### Bug Fixes

* apply base CSV during bundle generation ([5df14bd](https://github.com/rhobs/monitoring-stack-operator/commit/5df14bd01e8403718c4f67229e69ace61fed8663))
* parametrize the namespace of the prometheus operator ([5210561](https://github.com/rhobs/monitoring-stack-operator/commit/5210561f812b88c8eba1089f568d5908dc3e9cf9))


### [0.0.2](https://github.com/rhobs/monitoring-stack-operator/commit/) (2021-10-13)


### Bug Fixes

* installation of kustomize tool ([#20](https://github.com/rhobs/monitoring-stack-operator/pull/20)) ([96f5221](https://github.com/rhobs/monitoring-stack-operator/commit/96f52217928aff29746edbd520693d66248e161a))
