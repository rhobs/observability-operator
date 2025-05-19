# Changelog

All notable changes to this project will be documented in this file. See [standard-version](https://github.com/conventional-changelog/standard-version) for commit guidelines.

## [1.2.0](https://github.com/rhobs/monitoring-stack-operator/commit/) (2025-05-19)


### Features

*  Add Perses Flag OU-571 ([#664](https://github.com/rhobs/monitoring-stack-operator/issues/664)) ([c5b39c1](https://github.com/rhobs/monitoring-stack-operator/commit/c5b39c12719896be405f53916c6010d2b500fe13))
* add Incidents backend ([#678](https://github.com/rhobs/monitoring-stack-operator/issues/678)) ([5bcf2db](https://github.com/rhobs/monitoring-stack-operator/commit/5bcf2db7e316faa81fc9143d14b1f75b6fb8a4f5))
* add PatternFly 5 stream for distributed tracing UI plugin ([#736](https://github.com/rhobs/monitoring-stack-operator/issues/736)) ([b3176ac](https://github.com/rhobs/monitoring-stack-operator/commit/b3176ac82e82ba803ff1a6f704534c65a17a5725))
* add perses operator deployment and apis ([#680](https://github.com/rhobs/monitoring-stack-operator/issues/680)) ([5ba0330](https://github.com/rhobs/monitoring-stack-operator/commit/5ba03304119eb2545e277e55bc8cea1441c23d24))
* add schema configuration for logging view plugin ([#733](https://github.com/rhobs/monitoring-stack-operator/issues/733)) ([0124fd8](https://github.com/rhobs/monitoring-stack-operator/commit/0124fd89cab3a6b122e484dd90d396a9b9855ee1))
* add support for multiple images in plugins ([#689](https://github.com/rhobs/monitoring-stack-operator/issues/689)) ([e42e8a8](https://github.com/rhobs/monitoring-stack-operator/commit/e42e8a8e514a5b95e47842148496089e8336ce33))
* deploying COO in place of OBO on SC clusters ([#667](https://github.com/rhobs/monitoring-stack-operator/issues/667)) ([c9eb0e0](https://github.com/rhobs/monitoring-stack-operator/commit/c9eb0e0f1d16d27a13b10d9e2db5fe281e790f0b))
* deploying COO in place of OBO on SC clusters ([#716](https://github.com/rhobs/monitoring-stack-operator/issues/716)) ([dfe4e21](https://github.com/rhobs/monitoring-stack-operator/commit/dfe4e21cefceb2707c88da2faf00f005607844e6))
* discover default lokistack ([#690](https://github.com/rhobs/monitoring-stack-operator/issues/690)) ([648c091](https://github.com/rhobs/monitoring-stack-operator/commit/648c0910237e3491e95b6ddcc77ebb98a0645048))
* drop support for ui monitoring 4.14 ([#688](https://github.com/rhobs/monitoring-stack-operator/issues/688)) ([cd66c38](https://github.com/rhobs/monitoring-stack-operator/commit/cd66c380ea9876c2b3caf1bf5ea2e487fde786e3))
* enable pprof on the operator ([#725](https://github.com/rhobs/monitoring-stack-operator/issues/725)) ([2efed3c](https://github.com/rhobs/monitoring-stack-operator/commit/2efed3cb55452bd946fd87cdc86472a9667462ce))
* log Kubernetes events related to TLS certificate ([#637](https://github.com/rhobs/monitoring-stack-operator/issues/637)) ([a0ff1f3](https://github.com/rhobs/monitoring-stack-operator/commit/a0ff1f3133fb8464af3983518a0cc1d384ebb2e3))
* remove acm-version check for acm-alerting ui feature ([#677](https://github.com/rhobs/monitoring-stack-operator/issues/677)) ([365e30d](https://github.com/rhobs/monitoring-stack-operator/commit/365e30d3392c9ec3114bbc1d1eab3e46c86468e6))
* TLS support for the Thanos web endpoint ([#598](https://github.com/rhobs/monitoring-stack-operator/issues/598)) ([d989f9a](https://github.com/rhobs/monitoring-stack-operator/commit/d989f9ac839c196efc6d8d0146588d780a08ba88))
* update health-analyzer to v0.5.0 ([#738](https://github.com/rhobs/monitoring-stack-operator/issues/738)) ([7831111](https://github.com/rhobs/monitoring-stack-operator/commit/7831111b57f19c2765b2c7916c1a6f468346fcc1))
* update UIPlugin images to prep  for COO 1.1 release ([#687](https://github.com/rhobs/monitoring-stack-operator/issues/687)) ([2b6fe6d](https://github.com/rhobs/monitoring-stack-operator/commit/2b6fe6d924843858b1f1d581a9917f5cdd16f182))


### Bug Fixes

* default image for distributed tracing new stream ([#737](https://github.com/rhobs/monitoring-stack-operator/issues/737)) ([be7ed12](https://github.com/rhobs/monitoring-stack-operator/commit/be7ed128aa31bff628ee831e905f4dfc69dc1dc6))
* failed to get metric when install upstream ObO on OCP ([#654](https://github.com/rhobs/monitoring-stack-operator/issues/654)) ([6d7b707](https://github.com/rhobs/monitoring-stack-operator/commit/6d7b70724364dcc5996d3e174ca32414fb9db0b3))
* mktemp command expects a template string that includes a minimum of six X characters on some os ([#650](https://github.com/rhobs/monitoring-stack-operator/issues/650)) ([21f303b](https://github.com/rhobs/monitoring-stack-operator/commit/21f303b0b9a33b2f28a877824006bd5d57bc71ea))
* test script failure when enable OCP feature gate ([#695](https://github.com/rhobs/monitoring-stack-operator/issues/695)) ([79cbdf6](https://github.com/rhobs/monitoring-stack-operator/commit/79cbdf6efb493ccdb467fe778cc20b543e5cc17b))
* update docs with new monitoring UIPlugin schema ([#681](https://github.com/rhobs/monitoring-stack-operator/issues/681)) ([6e9607a](https://github.com/rhobs/monitoring-stack-operator/commit/6e9607a6d6d39298672b606c9abbf5c502b35363))
* web UI case failure in downstream ([#719](https://github.com/rhobs/monitoring-stack-operator/issues/719)) ([08ec7ed](https://github.com/rhobs/monitoring-stack-operator/commit/08ec7ed87e8b23761b5f6d47f773c66333365156))

## [1.0.0](https://github.com/rhobs/monitoring-stack-operator/commit/) (2024-12-11)


### Features

* add "Enable cluster monitoring" checkbox to OCP console ([#628](https://github.com/rhobs/monitoring-stack-operator/issues/628)) ([490a91d](https://github.com/rhobs/monitoring-stack-operator/commit/490a91d59fe604ff7c45bb5ad93da8400bd58903))
* add basic gather script for monitoring components ([#614](https://github.com/rhobs/monitoring-stack-operator/issues/614)) ([75dbd91](https://github.com/rhobs/monitoring-stack-operator/commit/75dbd91af06044d963df10d2987399ea425ddd4d))
* add monitoring-plugin uiplugin ([#575](https://github.com/rhobs/monitoring-stack-operator/issues/575)) ([48df915](https://github.com/rhobs/monitoring-stack-operator/commit/48df9151aeb735716f9c1a9fc18bf62f7de5b67d))
* add operator controller to add ServiceMonitor ([#616](https://github.com/rhobs/monitoring-stack-operator/issues/616)) ([5f2b6e5](https://github.com/rhobs/monitoring-stack-operator/commit/5f2b6e5eb86a4186d80ae56d5957606118d10cf0))
* add support annotation to uiplugins at runtime ([#604](https://github.com/rhobs/monitoring-stack-operator/issues/604)) ([41c578c](https://github.com/rhobs/monitoring-stack-operator/commit/41c578c8362d60f90736057e51f3282ec297f23f))
* add Trace support in korrel8r ([#597](https://github.com/rhobs/monitoring-stack-operator/issues/597)) ([a74b4a5](https://github.com/rhobs/monitoring-stack-operator/commit/a74b4a5d71eda5328673fc4dc2f3177865578dab))
* deploy PrometheusRule resource for the operator ([#629](https://github.com/rhobs/monitoring-stack-operator/issues/629)) ([a63fe12](https://github.com/rhobs/monitoring-stack-operator/commit/a63fe12ac4e112103d6a110e8fe89e164f0b051e))
* enable HTTPS in OpenShift clusters ([#595](https://github.com/rhobs/monitoring-stack-operator/issues/595)) ([a826c09](https://github.com/rhobs/monitoring-stack-operator/commit/a826c093c99e29444c3c392c9438b905978443f8))
* require TLS client certificate for /metrics ([#611](https://github.com/rhobs/monitoring-stack-operator/issues/611)) ([8cb5a11](https://github.com/rhobs/monitoring-stack-operator/commit/8cb5a11f84b01c997df0d36de5dff5576df65480))


### Bug Fixes

* add finalizers to cleanup cluster scoped resources on stack deletion ([#608](https://github.com/rhobs/monitoring-stack-operator/issues/608)) ([b243203](https://github.com/rhobs/monitoring-stack-operator/commit/b24320335991995cb0e57ad9fd80869b317e9360))
* allow operator SA to create/update events ([#623](https://github.com/rhobs/monitoring-stack-operator/issues/623)) ([6cae01e](https://github.com/rhobs/monitoring-stack-operator/commit/6cae01e11decc5e1df12948af2099ff2e9aa534b))
* give operator controller a name ([#618](https://github.com/rhobs/monitoring-stack-operator/issues/618)) ([21e89a0](https://github.com/rhobs/monitoring-stack-operator/commit/21e89a01f57f9d971b080378ebc485974ef49610))
* hide Prometheus operator CRDs in UI ([#605](https://github.com/rhobs/monitoring-stack-operator/issues/605)) ([ab4338f](https://github.com/rhobs/monitoring-stack-operator/commit/ab4338fe7db8dcbf800f450fe407297a14fd48dd))
* must-gather collection scripts should be executable ([#640](https://github.com/rhobs/monitoring-stack-operator/issues/640)) ([a130a82](https://github.com/rhobs/monitoring-stack-operator/commit/a130a82fcd9ca8839c97ab6a8313a50e3e7b17e3))
* register scheme in operator ([#619](https://github.com/rhobs/monitoring-stack-operator/issues/619)) ([d86ab09](https://github.com/rhobs/monitoring-stack-operator/commit/d86ab09377f657acf3fd73e7ee4b6d51c10d3508))

### [0.4.2](https://github.com/rhobs/monitoring-stack-operator/commit/) (2024-10-08)


### Bug Fixes

* use expected plugin console names ([#581](https://github.com/rhobs/monitoring-stack-operator/issues/581)) ([3dbb0f6](https://github.com/rhobs/monitoring-stack-operator/commit/3dbb0f6edfb3d4f76aea7d67b40fa6a7167c3b1b))

### [0.4.1](https://github.com/rhobs/monitoring-stack-operator/commit/) (2024-10-02)


### Features

* TLS support for the Alertmanager web endpoint ([#495](https://github.com/rhobs/monitoring-stack-operator/issues/495)) ([a6f1387](https://github.com/rhobs/monitoring-stack-operator/commit/a6f13878a5273cc75d4d3b5189fa36565f2617f3))
* TLS support for the Prometheus web endpoint ([#492](https://github.com/rhobs/monitoring-stack-operator/issues/492)) ([1b494d1](https://github.com/rhobs/monitoring-stack-operator/commit/1b494d17ebaea9f78df772208dc62462820fa53d))


### Bug Fixes

* add finalizer to cleanup the console after uiplugin is deleted ([#576](https://github.com/rhobs/monitoring-stack-operator/issues/576)) ([4ce18d9](https://github.com/rhobs/monitoring-stack-operator/commit/4ce18d98ec93542f1bea400ec320f66bf2ceeaa3))

## [0.4.0](https://github.com/rhobs/monitoring-stack-operator/commit/) (2024-08-29)


### Features

* update RBAC to include listing TempoMonolithic CRs ([#545](https://github.com/rhobs/monitoring-stack-operator/issues/545)) ([3aa199d](https://github.com/rhobs/monitoring-stack-operator/commit/3aa199d90d16143f02e96006b7ec0cf8688680f2))

### [0.3.5](https://github.com/rhobs/monitoring-stack-operator/commit/) (2024-08-07)


### Features

* COO-261: Support tolerations and nodeselector in Monitoringstack ([#540](https://github.com/rhobs/monitoring-stack-operator/issues/540)) ([3f6d6eb](https://github.com/rhobs/monitoring-stack-operator/commit/3f6d6ebc78adc3f6b7be3adc47784b6e838e3b96))


### Bug Fixes

* adjust max version semver comparission to avoid excluding valid cluster versions ([#541](https://github.com/rhobs/monitoring-stack-operator/issues/541)) ([daee156](https://github.com/rhobs/monitoring-stack-operator/commit/daee15657d59a34daa178c4c1e1d80a8f34279b2))

### [0.3.4](https://github.com/rhobs/monitoring-stack-operator/commit/) (2024-07-29)


### Bug Fixes

* add support for ConsolePlugin v1 ([#530](https://github.com/rhobs/monitoring-stack-operator/issues/530)) ([dc492f6](https://github.com/rhobs/monitoring-stack-operator/commit/dc492f6fefcbe3a5b3388960c3351d7d75f8ed44))
* bump obo-prometheus-operator to v0.75.2-rhobs ([#537](https://github.com/rhobs/monitoring-stack-operator/issues/537)) ([386a780](https://github.com/rhobs/monitoring-stack-operator/commit/386a7808b47576f8179c8b0c152d603a7c42e95a))

### [0.3.3](https://github.com/rhobs/monitoring-stack-operator/commit/) (2024-06-28)


### Features

* restrict UIPlugin CRD names to allow a single instance per type ([#481](https://github.com/rhobs/monitoring-stack-operator/issues/481)) ([62c1920](https://github.com/rhobs/monitoring-stack-operator/commit/62c19207cdbe085b6dc2ca274f08bd3fcb403f92))


### Bug Fixes

* add Clusterrole to allow Korrel8r to view Logs and Metrics ([#517](https://github.com/rhobs/monitoring-stack-operator/issues/517)) ([0d7afff](https://github.com/rhobs/monitoring-stack-operator/commit/0d7afff739cf7129708281d6b782713b8735f90b))
* return the correct loki service names ([#521](https://github.com/rhobs/monitoring-stack-operator/issues/521)) ([351ead5](https://github.com/rhobs/monitoring-stack-operator/commit/351ead585f6f3fb8761df910c07c1b1d60faeeda))

### [0.3.2](https://github.com/rhobs/monitoring-stack-operator/commit/) (2024-06-17)


### Features

* bump Prometheus operator to v0.74.0 ([#518](https://github.com/rhobs/monitoring-stack-operator/issues/518)) ([abd2f03](https://github.com/rhobs/monitoring-stack-operator/commit/abd2f03741eb2d1de696136f0161557c09d5a7e7))


### Bug Fixes

* add korrel8r proxy configuration to logging view plugin ([#515](https://github.com/rhobs/monitoring-stack-operator/issues/515)) ([a1f0de3](https://github.com/rhobs/monitoring-stack-operator/commit/a1f0de357c38e45849f0ce33fdff3b0ef71917a2))

### [0.3.1](https://github.com/rhobs/monitoring-stack-operator/commit/) (2024-06-13)


### Bug Fixes

* shellcheck wrong curl ([#510](https://github.com/rhobs/monitoring-stack-operator/issues/510)) ([250dcbc](https://github.com/rhobs/monitoring-stack-operator/commit/250dcbce715df203905f14344be1e9acbc2884b4))

## [0.3.0](https://github.com/rhobs/monitoring-stack-operator/commit/) (2024-06-10)


### Features

* add distributed tracing and troubleshooting panel uiplugins ([#480](https://github.com/rhobs/monitoring-stack-operator/issues/480)) ([db0b62f](https://github.com/rhobs/monitoring-stack-operator/commit/db0b62f8be16d4ff549abd789c3f26218342590e))
* add Korrel8r plugin to UITroubleshootPanel ([#497](https://github.com/rhobs/monitoring-stack-operator/issues/497)) ([02690af](https://github.com/rhobs/monitoring-stack-operator/commit/02690afbee4d3b2bd9daf2e1aa4674ee29def5e2))
* support logging as ui-plugin ([#477](https://github.com/rhobs/monitoring-stack-operator/issues/477)) ([9ecc7fc](https://github.com/rhobs/monitoring-stack-operator/commit/9ecc7fcd21c305296b7ee82cbcb71f6345fb950a))


### Bug Fixes

* bind service with correct container port for korrel8r ([#504](https://github.com/rhobs/monitoring-stack-operator/issues/504)) ([4edf774](https://github.com/rhobs/monitoring-stack-operator/commit/4edf774d2044b9fd0d12ee792a0bcb170e2fad40))
* compatibility matrix unit tests ([#499](https://github.com/rhobs/monitoring-stack-operator/issues/499)) ([f3c6a61](https://github.com/rhobs/monitoring-stack-operator/commit/f3c6a61051c44198999e681cffe77183d1b77a14))
* compatibility matrix version validation ([#501](https://github.com/rhobs/monitoring-stack-operator/issues/501)) ([facefdc](https://github.com/rhobs/monitoring-stack-operator/commit/facefdcb72f8cfc16843663e1a25a702d7170c56))
* default goal in makefile and add goal for unit tests ([#475](https://github.com/rhobs/monitoring-stack-operator/issues/475)) ([b6ed9c5](https://github.com/rhobs/monitoring-stack-operator/commit/b6ed9c5f718d1242929157d950e908650799f892))
* duplicate monitoringstack name caused case unstable ([#478](https://github.com/rhobs/monitoring-stack-operator/issues/478)) ([6d91f2e](https://github.com/rhobs/monitoring-stack-operator/commit/6d91f2e576966c38e07b5647a7f2f45d67e3bbbc))
* fix UIPLugin console registration to avoid mutating existing cluster configuration ([#503](https://github.com/rhobs/monitoring-stack-operator/issues/503)) ([414e4f5](https://github.com/rhobs/monitoring-stack-operator/commit/414e4f565818775a1961ddaa5ea695d17b14d00f))
* include service proxy in distributed_tracing.go ([#502](https://github.com/rhobs/monitoring-stack-operator/issues/502)) ([d359118](https://github.com/rhobs/monitoring-stack-operator/commit/d3591180366b8b4ec2f0da05a01c459b35cfe6e5))
* install shellcheck for lint target ([#493](https://github.com/rhobs/monitoring-stack-operator/issues/493)) ([3b6f58c](https://github.com/rhobs/monitoring-stack-operator/commit/3b6f58c81e615e9fcc4c6267ae02ee7f6a8b76f1))
* null pointer error of case NoOwnerRefInvalidNamespaceReasonEvent ([#479](https://github.com/rhobs/monitoring-stack-operator/issues/479)) ([646a8ba](https://github.com/rhobs/monitoring-stack-operator/commit/646a8ba6df5e92f2abcd87393bce71f1dee39b1a))
* prevent other plugin types from using tracing and troubleshooting configurations ([#498](https://github.com/rhobs/monitoring-stack-operator/issues/498)) ([07ae4ef](https://github.com/rhobs/monitoring-stack-operator/commit/07ae4efe3d6684209ebd950ebb0951ae65f082eb))
* prevent reconcile loop for troubleshooting panel uiplugin ([#505](https://github.com/rhobs/monitoring-stack-operator/issues/505)) ([994ad0b](https://github.com/rhobs/monitoring-stack-operator/commit/994ad0b42e3ef86aea55d99464ab528a6a75e3d7))
* remove duplicate target in kustomize configuration ([#476](https://github.com/rhobs/monitoring-stack-operator/issues/476)) ([06027cf](https://github.com/rhobs/monitoring-stack-operator/commit/06027cfdb40f8e1dba9e8b13be677781da387c0f))
* Some typos and reconciliation of optional UIPlugin components ([#491](https://github.com/rhobs/monitoring-stack-operator/issues/491)) ([09dd760](https://github.com/rhobs/monitoring-stack-operator/commit/09dd760df805879f9d3967f3f769975ae9cbfa0d))

## [0.2.0](https://github.com/rhobs/monitoring-stack-operator/commit/) (2024-04-22)


### Features

* Add ObservabilityUI plugins API ([#434](https://github.com/rhobs/monitoring-stack-operator/issues/434)) ([92bae83](https://github.com/rhobs/monitoring-stack-operator/commit/92bae83b9a43c2080134452b240eba5ff15e12b7))
* make Thanos querier compliant with restricted policy ([#452](https://github.com/rhobs/monitoring-stack-operator/issues/452)) ([cd8cd42](https://github.com/rhobs/monitoring-stack-operator/commit/cd8cd4241d3cc464243deae2d21dff20e6d7a968))
* provide api option to enable otlp/http receiver ([#450](https://github.com/rhobs/monitoring-stack-operator/issues/450)) ([65ea6bd](https://github.com/rhobs/monitoring-stack-operator/commit/65ea6bdd37b4c19db97da43afd3d72e8eee1c843))


### Bug Fixes

* remove invalid owner ref on cluster role ([#460](https://github.com/rhobs/monitoring-stack-operator/issues/460)) ([fc12c57](https://github.com/rhobs/monitoring-stack-operator/commit/fc12c57af2af014d97fc0a01fe850b6b64940da6))

## [0.1.0](https://github.com/rhobs/monitoring-stack-operator/commit/) (2024-03-13)


### Bug Fixes

* use thanos default port in service and containerPort ([#414](https://github.com/rhobs/monitoring-stack-operator/issues/414)) ([2d6c82b](https://github.com/rhobs/monitoring-stack-operator/commit/2d6c82b9ff44a197a6ce06d2001b32570b61f376))

### [0.0.30](https://github.com/rhobs/monitoring-stack-operator/commit/) (2024-01-23)


### Bug Fixes

* update versions in bundle generation ([#411](https://github.com/rhobs/monitoring-stack-operator/issues/411)) ([755caba](https://github.com/rhobs/monitoring-stack-operator/commit/755caba8f0dc11940927b6b829ae176384a6227e))

### [0.0.29](https://github.com/rhobs/monitoring-stack-operator/commit/) (2024-01-22)


### Bug Fixes

* **test:** update ocp test scripts adding ci mode ([#398](https://github.com/rhobs/monitoring-stack-operator/issues/398)) ([6f2c229](https://github.com/rhobs/monitoring-stack-operator/commit/6f2c2293313962619134b38d1dd78ce39c098831))

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
