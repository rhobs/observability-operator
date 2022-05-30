# Changelog

All notable changes to this project will be documented in this file. See [standard-version](https://github.com/conventional-changelog/standard-version) for commit guidelines.

### [0.0.9](https://github.com/rhobs/monitoring-stack-operator/tree/v0.0.9) (2022-05-30)


### Features

* Update prometheus-operator dependency to 0.55.1 ([#140](https://github.com/rhobs/monitoring-stack-operator/issues/140)) ([fd6b78c](https://github.com/rhobs/monitoring-stack-operator/commit/fd6b78c5faeb45c02551edb36ac139754c68ac07))

### [0.0.8](https://github.com/rhobs/monitoring-stack-operator/tree/v0.0.8) (2022-05-17)


### ⚠ BREAKING CHANGES

* [MON-2247](https://issues.redhat.com/browse/MON-2247): MSO does not deploy grafana operator anymore.

### Features

* Add remotewrite option to PrometheusConfig ([b6319a6](https://github.com/rhobs/monitoring-stack-operator/commit/b6319a62a7e8102daa0870d1c4413a3fa1cbe857))
* Add support for configuring Prometheus external labels ([#126](https://github.com/rhobs/monitoring-stack-operator/issues/126)) ([02289d1](https://github.com/rhobs/monitoring-stack-operator/commit/02289d1854c96afc68bb2a2389df228ad586ff3d)), closes [#125](https://github.com/rhobs/monitoring-stack-operator/issues/125)
* Runs Prometheus in HA mode by default ([cdf8ce4](https://github.com/rhobs/monitoring-stack-operator/commit/cdf8ce46ae70238c32835ac9a2e0d8df8a7926d7))
* Removes the grafana operator ([2f5ed6d](https://github.com/rhobs/monitoring-stack-operator/commit/2f5ed6d34df4f9310205ebfa6f92f9e92dc5f58e))

### [0.0.7](https://github.com/rhobs/monitoring-stack-operator/commit/) (2022-04-06)


### Features

* add a pod disruption budget for Alertmanager ([94db768](https://github.com/rhobs/monitoring-stack-operator/commit/94db768d28c6f3cdaa679f2ee958a440be001df0))
* add alert rules for mso ([#94](https://github.com/rhobs/monitoring-stack-operator/issues/94)) ([c13d605](https://github.com/rhobs/monitoring-stack-operator/commit/c13d605bc108c71cbca1b83e57502431dd8c9c2f))
* enable persistent storage for prometheus ([#111](https://github.com/rhobs/monitoring-stack-operator/issues/111)) ([b68b750](https://github.com/rhobs/monitoring-stack-operator/commit/b68b7503e7dc84f083ccd25c73a33ef5da1fae6a))
* deploy alertmanagers on different nodes ([79fad13](https://github.com/rhobs/monitoring-stack-operator/commit/79fad138f46c6f5fa6c04bbcc54cadcbfc234e34))
* upgrade grafana-operator to 4.1.0 ([3a741ee](https://github.com/rhobs/monitoring-stack-operator/commit/3a741ee45e7d7c72cc8fe76ac2a01e6b144d5434))

### Bug Fixes

* clean up grafana datasource on deleting monitoring stack ([#107](https://github.com/rhobs/monitoring-stack-operator/issues/107)) ([b125c25](https://github.com/rhobs/monitoring-stack-operator/commit/b125c25d4131dd6219d2f112113c3c5b886188fe))
* deleted grafana datasources will now be recreated ([#96](https://github.com/rhobs/monitoring-stack-operator/issues/96)) ([2c71d1d](https://github.com/rhobs/monitoring-stack-operator/commit/2c71d1d27682ef546b2a219d21fbce9afddc0231))
* establish a watch on Grafana CRs only after the CRD is present ([f5787ed](https://github.com/rhobs/monitoring-stack-operator/commit/f5787ed2b540058fec3e741ce7f43c4c440f2f31))
* grafana-operator now uses more optimised watches ([7b1cd05](https://github.com/rhobs/monitoring-stack-operator/commit/7b1cd05ded3d3f556df5ff9a7fd0c97e1c494c92))
* increase resource(memory) limit of mso operator ([dd0fd92](https://github.com/rhobs/monitoring-stack-operator/commit/dd0fd9201de789a0131fd9118251e55afbdef9ec))
* fix install-plan approval logic to approve the right plan ([#97](https://github.com/rhobs/monitoring-stack-operator/issues/97)) ([b669e08](https://github.com/rhobs/monitoring-stack-operator/commit/b669e086cdf9e7a156d9ebece99b296e260e41a2))

### [0.0.6](https://github.com/rhobs/monitoring-stack-operator/commit/) (2021-12-02)


### Bug Fixes

* fix self-scrape prometheus configuration for stacks ([f34c8bf](https://github.com/rhobs/monitoring-stack-operator/commit/f34c8bf9a0c407679d1315c21380c4b4caf3cf8c))
* prevent automatic upgrades of Grafana Operator ([44009d7](https://github.com/rhobs/monitoring-stack-operator/commit/44009d7ff652ba6d530a2d595b286aaaf0afa2bb))

### [0.0.5](https://github.com/rhobs/monitoring-stack-operator/commit/) (2021-11-29)


### Features

* make the module go-gettable ([94342b7](https://github.com/rhobs/monitoring-stack-operator/commit/94342b772c886971cd9e3b52c652efadda65bc86))
* update prometheus-operator to 0.52.1 ([c637521](https://github.com/rhobs/monitoring-stack-operator/commit/c6375218b342abf98406bcaa5043452ff85a4ca2))

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
