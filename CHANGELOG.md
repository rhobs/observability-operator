# Changelog

All notable changes to this project will be documented in this file. See [standard-version](https://github.com/conventional-changelog/standard-version) for commit guidelines.

### [0.0.28](https://github.com/rhobs/monitoring-stack-operator/commit/) (2023-11-08)

### [0.0.27](https://github.com/rhobs/monitoring-stack-operator/commit/) (2023-11-07)


### Features

* bump Prometheus operator to v0.68.0 ([#371](https://github.com/rhobs/monitoring-stack-operator/issues/371)) ([50b7889](https://github.com/rhobs/monitoring-stack-operator/commit/50b7889c87005da2c240b88dbf102893eff77117))
* bump Prometheus operator to v0.69.0 ([#380](https://github.com/rhobs/monitoring-stack-operator/issues/380)) ([7facafd](https://github.com/rhobs/monitoring-stack-operator/commit/7facafdee90820c70b500047b1a93edffaa9fe96))

### [0.0.26](https://github.com/rhobs/monitoring-stack-operator/commit/) (2023-10-11)


### Features

* Bump PO to v0.66.0 ([#319](https://github.com/rhobs/monitoring-stack-operator/issues/319)) ([5e42a1d](https://github.com/rhobs/monitoring-stack-operator/commit/5e42a1dc29027166c5fd2e75894bf53cd0be710b))


### Bug Fixes

* add description field to CSV base ([#366](https://github.com/rhobs/monitoring-stack-operator/issues/366)) ([22bb08b](https://github.com/rhobs/monitoring-stack-operator/commit/22bb08b5ca8b7e12d4974a75c412a063a6658789))
* Clean up deprecated functions  ([#326](https://github.com/rhobs/monitoring-stack-operator/issues/326)) ([3f29722](https://github.com/rhobs/monitoring-stack-operator/commit/3f29722f49b1cd836f1ff0fc05395d985ca1d586))
* remove ServiceMonitor observability-operator from bundle ([#354](https://github.com/rhobs/monitoring-stack-operator/issues/354)) ([e9f13ce](https://github.com/rhobs/monitoring-stack-operator/commit/e9f13ce9901205f94dc825033673351be81bf5a3))
* remove stripped down crds hack ([#362](https://github.com/rhobs/monitoring-stack-operator/issues/362)) ([4f1dc2f](https://github.com/rhobs/monitoring-stack-operator/commit/4f1dc2f0c3e6bfb14aa914969f98cc8c9c082575))
* test scripts and readme doc about uninstallation ([#330](https://github.com/rhobs/monitoring-stack-operator/issues/330)) ([fca1667](https://github.com/rhobs/monitoring-stack-operator/commit/fca16679cb6cb10bf10a7731e046b11617572711))
* update github workflow to use node>=16 ([#336](https://github.com/rhobs/monitoring-stack-operator/issues/336)) ([a66295f](https://github.com/rhobs/monitoring-stack-operator/commit/a66295f0e2c096cd33b522310de7e6f3ca76e7f2))
* use framework default timeout in ns tests ([#335](https://github.com/rhobs/monitoring-stack-operator/issues/335)) ([d19d7f2](https://github.com/rhobs/monitoring-stack-operator/commit/d19d7f2eb8a76f9d51a6291de1b36bedcd17ddc4))

### [0.0.25](https://github.com/rhobs/monitoring-stack-operator/commit/) (2023-08-07)


### Bug Fixes

* add node tolerations to deployments ([#321](https://github.com/rhobs/monitoring-stack-operator/issues/321)) ([d0ba3a9](https://github.com/rhobs/monitoring-stack-operator/commit/d0ba3a92fbdb8e82033102f1302b31115d3448c8))
* test case multi-namespace_support ([#312](https://github.com/rhobs/monitoring-stack-operator/issues/312)) ([6c09f46](https://github.com/rhobs/monitoring-stack-operator/commit/6c09f466c76790909b768f97b950b56af4b05608))

### [0.0.24](https://github.com/rhobs/monitoring-stack-operator/commit/) (2023-07-27)


### Bug Fixes

* enforce scheduling registry pod onto infra nodes ([#313](https://github.com/rhobs/monitoring-stack-operator/issues/313)) ([23a60b4](https://github.com/rhobs/monitoring-stack-operator/commit/23a60b4789520c3c9087c1987a28fa1d637578b7))

### [0.0.23](https://github.com/rhobs/monitoring-stack-operator/commit/) (2023-07-11)


### Bug Fixes

* add repo details to bundle ([#303](https://github.com/rhobs/monitoring-stack-operator/issues/303)) ([687e0cb](https://github.com/rhobs/monitoring-stack-operator/commit/687e0cbb50421b3f74fbeb32e40e56b321e5af9e))
* broken release candidate workflow ([#306](https://github.com/rhobs/monitoring-stack-operator/issues/306)) ([56f9e2c](https://github.com/rhobs/monitoring-stack-operator/commit/56f9e2c03638aa88908d1d223cb2ae5e981fc9cf))
* **doc:** use right terminology in release doc ([f65d0d2](https://github.com/rhobs/monitoring-stack-operator/commit/f65d0d248207a89e7a6bc72e5b235fd1c95c0c38))
* make catalogsource compatible with restricted SCC enforcement ([d0d4c74](https://github.com/rhobs/monitoring-stack-operator/commit/d0d4c748eb7426815525f5e283dbd190175c6d21))

### [0.0.22](https://github.com/rhobs/monitoring-stack-operator/commit/) (2023-06-04)


### Features

* add probe and scrapeConfig selectors a2f0897
* make operator publishable to openshift community-catalog ([#295](https://github.com/rhobs/monitoring-stack-operator/issues/295)) 5e0f6c3


### Bug Fixes

* ensure OLM bundle installs fine all supported OpenShift Versions ([#299](https://github.com/rhobs/monitoring-stack-operator/issues/299)) e33f901
* **test:** ensure test report follows osde2e recommendation ([#296](https://github.com/rhobs/monitoring-stack-operator/issues/296)) 6ef4b1e
* update url link to rhobs-handbook.netlify.app ([#289](https://github.com/rhobs/monitoring-stack-operator/issues/289)) de3e98d, closes #287

### [0.0.21](https://github.com/rhobs/monitoring-stack-operator/commit/) (2023-05-23)


### Features

* add scrape interval parameter to prometheus config 40de685
* bumpup Prometheus Operator to 0.65.1 82fc388


### Bug Fixes

* e2e test cleanup and exit code 590b339
* field manager name for generated resources 853f04f
* **test:** use OPERATORS_NS instead of hardcoded namespace 1cadc70
* update correct operator version in CSV 60c7be6
* wrong catalog sourcename in k8s subscription b7e4b57

### [0.0.20](https://github.com/rhobs/monitoring-stack-operator/commit/) (2023-01-16)


### Features

* add resource variables to SyncSelectorSet template ([#247](https://github.com/rhobs/monitoring-stack-operator/issues/247)) ([68fd127](https://github.com/rhobs/monitoring-stack-operator/commit/68fd12740504382ac9ad3431e735f69bf35f555e))


### Bug Fixes

* rename alert names to follow convention ([#246](https://github.com/rhobs/monitoring-stack-operator/issues/246)) ([c2ecb85](https://github.com/rhobs/monitoring-stack-operator/commit/c2ecb858856e68d1cf2f376b0b875410ef74d8ed))
* use mebibytes instead of megabytes for resource defaults ([#248](https://github.com/rhobs/monitoring-stack-operator/issues/248)) ([4a62425](https://github.com/rhobs/monitoring-stack-operator/commit/4a62425726d5b941609355574d21c80919d59deb))

### [0.0.19](https://github.com/rhobs/monitoring-stack-operator/commit/) (2023-01-10)

* this release only includes a change to the build process of the catalog image c188473

### [0.0.18](https://github.com/rhobs/monitoring-stack-operator/commit/) (2023-01-05)


### Features

* add resourceDiscovery status condition ([#223](https://github.com/rhobs/monitoring-stack-operator/issues/223)) ([1ea726d](https://github.com/rhobs/monitoring-stack-operator/commit/1ea726d628eab88a5a72d61e14f08aea14c7078e))
* upgrade PO to 0.61.0-rhobs1 ([#234](https://github.com/rhobs/monitoring-stack-operator/issues/234)) ([8f342e8](https://github.com/rhobs/monitoring-stack-operator/commit/8f342e8dae0c4ec50f58a3be75fa1660094002a2))

### [0.0.17](https://github.com/rhobs/monitoring-stack-operator/commit/) (2022-12-01)


### Features

* add support for watching multiple namespaces ([4bda99c](https://github.com/rhobs/monitoring-stack-operator/commit/4bda99c0c6dc4f5132dbc674abb4bf86eced4aa7))


### Bug Fixes

* number of Prometheus replicas can be ([87bd1f7](https://github.com/rhobs/monitoring-stack-operator/commit/87bd1f7141efe6da0259a71858e189895350cdc8))
* update log levels to reflect alertmanager levels ([#221](https://github.com/rhobs/monitoring-stack-operator/issues/221)) ([b71d145](https://github.com/rhobs/monitoring-stack-operator/commit/b71d1455c352a7f71681fa6dfabead3ed200b5e0))

### [0.0.16](https://github.com/rhobs/monitoring-stack-operator/tree/v0.0.16) (2022-11-02)

### Features

* deploy admission webhook 8cc57d7


### Bug Fixes

* hide internal CRDs from OLM UI 5c0f013
* release workflow broken due to invalid syntax 7ad0d8a
* target management clusters in SSS ([#207](https://github.com/rhobs/monitoring-stack-operator/issues/207)) b660849
* update stack status only if Prometheus generation is different 270ec28
* validate Prometheus replicas number  cbb95f3

### [0.0.15](https://github.com/rhobs/monitoring-stack-operator/tree/v0.0.15) (2022-10-13)


### Features

### ⚠ BREAKING CHANGES
* Switches from using platform Prometheus Operator to a forked PO based on 0.60.0  ([c1f534a](https://github.com/rhobs/monitoring-stack-operator/commit/c1f534a15b71c948c3c821af671207d1ac4f25f3))


### [0.0.14](https://github.com/rhobs/monitoring-stack-operator/commit/) (2022-09-20)


### Features

* add API option to disable Alertmanager deployment ([217eafc](https://github.com/rhobs/monitoring-stack-operator/commit/217eafcc78a956dcbd77fd81b3276b6c55f5ae26))
* add health probes to operator ([8661936](https://github.com/rhobs/monitoring-stack-operator/commit/86619360549364991adf48e5581113af3df48647))
* switch to file-based OLM catalogs ([#195](https://github.com/rhobs/monitoring-stack-operator/issues/195)) ([f3db3e2](https://github.com/rhobs/monitoring-stack-operator/commit/f3db3e2c21ac58d16a6aef07d7e8c9de34b286ff))


### Bug Fixes

* report Available=False condition when Prometheus is degraded ([ece8d8c](https://github.com/rhobs/monitoring-stack-operator/commit/ece8d8c16663d1b04221ac0a8284da44daded1e2))

### [0.0.13](https://github.com/rhobs/monitoring-stack-operator/tree/v0.0.13) (2022-07-26)

### Features

* add option to pass EnableRemoteWriteReceiver to Prometheus CR ([37c777e](https://github.com/rhobs/monitoring-stack-operator/commit/37c777e9bca860abcee3d36f9148da3d9f4aa47a))
* add status attribute to the MonitoringStack CRD ([#143](https://github.com/rhobs/monitoring-stack-operator/issues/143)) ([bcda150](https://github.com/rhobs/monitoring-stack-operator/commit/bcda15013a034dd646c8f7b94ceb17ebcd96c6dc))

### [0.0.12](https://github.com/rhobs/monitoring-stack-operator/tree/v0.0.12) (2022-07-08)

### Bug Fixes

* **olm:** fix error when subscribing due to missing index image ([#167](https://github.com/rhobs/monitoring-stack-operator/issues/167)) ([b7186d8](https://github.com/rhobs/monitoring-stack-operator/commit/b7186d87a490e6b195de0fc46fd8c993cbf60657))
* set resources for OO and P-O ([8658ccf](https://github.com/rhobs/monitoring-stack-operator/commit/8658ccfee334e7e1e9a3a361f54cce22227e92ab)), closes [#166](https://github.com/rhobs/monitoring-stack-operator/issues/166)

### [0.0.11](https://github.com/rhobs/monitoring-stack-operator/commit/) (2022-06-17)


### Features

* set soft affinity on operator deployments ([#152](https://github.com/rhobs/monitoring-stack-operator/issues/152)) ([f83e38b](https://github.com/rhobs/monitoring-stack-operator/commit/f83e38b5df749390a4212525ab601486c7e4c2da))
* update prometheus-operator dependency in go.mod ([#159](https://github.com/rhobs/monitoring-stack-operator/issues/159)) ([ff75353](https://github.com/rhobs/monitoring-stack-operator/commit/ff75353ef68dab0a0892dacd02d524c56f4ea705))


### Bug Fixes

* change slack details in README according to rename ([#155](https://github.com/rhobs/monitoring-stack-operator/issues/155)) ([be9fe46](https://github.com/rhobs/monitoring-stack-operator/commit/be9fe46072b869006b13333810d7ad2d492e4359))
* grants SA of components access to nonroot SCC ([#161](https://github.com/rhobs/monitoring-stack-operator/issues/161)) ([83567e0](https://github.com/rhobs/monitoring-stack-operator/commit/83567e0066b3bc8a04b5a859437f08ad1e477471))
* remove SeccompProfile ([#164](https://github.com/rhobs/monitoring-stack-operator/issues/164)) ([3098fc2](https://github.com/rhobs/monitoring-stack-operator/commit/3098fc20c8183268f0431c666d81cd2cd75ad6e0))
* rename operator catalog ([390a4aa](https://github.com/rhobs/monitoring-stack-operator/commit/390a4aa250e3c0d401c9bc0a68bce041f6a6df8b))
* set seccomp profiles and grant SAs necessary premissions to run ([#154](https://github.com/rhobs/monitoring-stack-operator/issues/154)) ([1d44825](https://github.com/rhobs/monitoring-stack-operator/commit/1d448254d7bfce836c260e5af7962de158af2f27))
* subscription source should be observability-operator ([ad8101a](https://github.com/rhobs/monitoring-stack-operator/commit/ad8101a93592906c406f478d5b857995eb52164e))

### [0.0.10](https://github.com/rhobs/monitoring-stack-operator/tree/0.0.10) (2022-06-01)

### ⚠ BREAKING CHANGES

* [ISSUE - 145](https://github.com/rhobs/observability-operator/issues/145)
    The Operator has been renamed to `Observability Operator`
* **NOTE:** The last release of Monitoring Stack Operator is `0.0.9`


### Migrating from 0.0.9

* Uninstall and unsubscribe the old Monitoring Stack Operator
* Subscribe to the new `Observability Operator` - see :
    ``hack/olm/catalog-src.yaml``

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
