# Change Log

## [0.8.0-rc.0](https://github.com/kubedb/elasticsearch/tree/0.8.0-rc.0) (2018-05-28)
[Full Changelog](https://github.com/kubedb/elasticsearch/compare/0.8.0-beta.2...0.8.0-rc.0)

**Merged pull requests:**

-  Initialize database heapsize from Resource.requests [\#138](https://github.com/kubedb/elasticsearch/pull/138) ([the-redback](https://github.com/the-redback))
- concourse [\#137](https://github.com/kubedb/elasticsearch/pull/137) ([tahsinrahman](https://github.com/tahsinrahman))
- Refactored E2E testing to support self-hosted operator with proper deployment configuration [\#136](https://github.com/kubedb/elasticsearch/pull/136) ([the-redback](https://github.com/the-redback))
- to allow Request header field Authorization [\#135](https://github.com/kubedb/elasticsearch/pull/135) ([aerokite](https://github.com/aerokite))
- Skip delete requests for empty resources [\#134](https://github.com/kubedb/elasticsearch/pull/134) ([the-redback](https://github.com/the-redback))
- Use separate resource & storage [\#133](https://github.com/kubedb/elasticsearch/pull/133) ([aerokite](https://github.com/aerokite))
- Don't panic if admission options is nil [\#132](https://github.com/kubedb/elasticsearch/pull/132) ([tamalsaha](https://github.com/tamalsaha))
- Disable admission controllers for webhook server [\#131](https://github.com/kubedb/elasticsearch/pull/131) ([tamalsaha](https://github.com/tamalsaha))
- Separate ApiGroup for Mutating and Validating webhook && upgraded osm to 0.7.0 [\#130](https://github.com/kubedb/elasticsearch/pull/130) ([the-redback](https://github.com/the-redback))
- Update client-go to 7.0.0 [\#129](https://github.com/kubedb/elasticsearch/pull/129) ([tamalsaha](https://github.com/tamalsaha))
- Bundle webhook server & Used  SharedInformer Factory with n-EventHandler [\#128](https://github.com/kubedb/elasticsearch/pull/128) ([the-redback](https://github.com/the-redback))
- Moved admission webhook packages into elasticsearch repo [\#127](https://github.com/kubedb/elasticsearch/pull/127) ([the-redback](https://github.com/the-redback))
- Add travis yaml [\#125](https://github.com/kubedb/elasticsearch/pull/125) ([tahsinrahman](https://github.com/tahsinrahman))

## [0.8.0-beta.2](https://github.com/kubedb/elasticsearch/tree/0.8.0-beta.2) (2018-02-27)
[Full Changelog](https://github.com/kubedb/elasticsearch/compare/0.8.0-beta.1...0.8.0-beta.2)

**Merged pull requests:**

- Use apps/v1 [\#123](https://github.com/kubedb/elasticsearch/pull/123) ([aerokite](https://github.com/aerokite))
- update validation [\#122](https://github.com/kubedb/elasticsearch/pull/122) ([aerokite](https://github.com/aerokite))
- Fix for pointer Type [\#121](https://github.com/kubedb/elasticsearch/pull/121) ([aerokite](https://github.com/aerokite))
- pass same type to Equal method [\#120](https://github.com/kubedb/elasticsearch/pull/120) ([aerokite](https://github.com/aerokite))
- Modify certificate DNS [\#119](https://github.com/kubedb/elasticsearch/pull/119) ([aerokite](https://github.com/aerokite))
- Fixed dormantdb matching & Raised throttling time & Fixed Elasticsearch version checking [\#118](https://github.com/kubedb/elasticsearch/pull/118) ([the-redback](https://github.com/the-redback))
- Use official code generator scripts [\#117](https://github.com/kubedb/elasticsearch/pull/117) ([tamalsaha](https://github.com/tamalsaha))
- Use github.com/pkg/errors [\#116](https://github.com/kubedb/elasticsearch/pull/116) ([tamalsaha](https://github.com/tamalsaha))
- Use separate certs for node & client and use random password by default [\#115](https://github.com/kubedb/elasticsearch/pull/115) ([aerokite](https://github.com/aerokite))
- Fix pluralization of Elasticsearch [\#114](https://github.com/kubedb/elasticsearch/pull/114) ([tamalsaha](https://github.com/tamalsaha))

## [0.8.0-beta.1](https://github.com/kubedb/elasticsearch/tree/0.8.0-beta.1) (2018-01-29)
[Full Changelog](https://github.com/kubedb/elasticsearch/compare/0.8.0-beta.0...0.8.0-beta.1)

**Merged pull requests:**

- Build image from other directories [\#113](https://github.com/kubedb/elasticsearch/pull/113) ([the-redback](https://github.com/the-redback))
- Fix for Job watcher [\#111](https://github.com/kubedb/elasticsearch/pull/111) ([aerokite](https://github.com/aerokite))
- reorg docker code structure [\#110](https://github.com/kubedb/elasticsearch/pull/110) ([aerokite](https://github.com/aerokite))

## [0.8.0-beta.0](https://github.com/kubedb/elasticsearch/tree/0.8.0-beta.0) (2018-01-07)
[Full Changelog](https://github.com/kubedb/elasticsearch/compare/0.7.1...0.8.0-beta.0)

**Merged pull requests:**

- Remove EnableRbac from Option [\#109](https://github.com/kubedb/elasticsearch/pull/109) ([aerokite](https://github.com/aerokite))
- pass analytics client-id as ENV [\#108](https://github.com/kubedb/elasticsearch/pull/108) ([aerokite](https://github.com/aerokite))
- update docker image validation [\#107](https://github.com/kubedb/elasticsearch/pull/107) ([aerokite](https://github.com/aerokite))
- Use work queue [\#106](https://github.com/kubedb/elasticsearch/pull/106) ([aerokite](https://github.com/aerokite))
- Reorg location of docker images [\#105](https://github.com/kubedb/elasticsearch/pull/105) ([aerokite](https://github.com/aerokite))
- Set client id for analytics [\#104](https://github.com/kubedb/elasticsearch/pull/104) ([tamalsaha](https://github.com/tamalsaha))
- Add explanation for oid bytes [\#103](https://github.com/kubedb/elasticsearch/pull/103) ([tamalsaha](https://github.com/tamalsaha))
- Revendor [\#102](https://github.com/kubedb/elasticsearch/pull/102) ([tamalsaha](https://github.com/tamalsaha))
- Various fixes in docker & controller [\#101](https://github.com/kubedb/elasticsearch/pull/101) ([aerokite](https://github.com/aerokite))
- Fix CRD registration [\#100](https://github.com/kubedb/elasticsearch/pull/100) ([the-redback](https://github.com/the-redback))
- Remove deleted appcode/log package [\#99](https://github.com/kubedb/elasticsearch/pull/99) ([tamalsaha](https://github.com/tamalsaha))
- Use monitoring tools from appscode/kutil [\#98](https://github.com/kubedb/elasticsearch/pull/98) ([tamalsaha](https://github.com/tamalsaha))
- Support elasticsearch 5.6.3 with dedicated nodes [\#97](https://github.com/kubedb/elasticsearch/pull/97) ([aerokite](https://github.com/aerokite))
- Use client-go 5.x [\#96](https://github.com/kubedb/elasticsearch/pull/96) ([tamalsaha](https://github.com/tamalsaha))

## [0.7.1](https://github.com/kubedb/elasticsearch/tree/0.7.1) (2017-10-04)
[Full Changelog](https://github.com/kubedb/elasticsearch/compare/0.7.0...0.7.1)

## [0.7.0](https://github.com/kubedb/elasticsearch/tree/0.7.0) (2017-09-26)
[Full Changelog](https://github.com/kubedb/elasticsearch/compare/0.6.0...0.7.0)

**Merged pull requests:**

- Set Affinity and Tolerations from CRD spec. [\#95](https://github.com/kubedb/elasticsearch/pull/95) ([tamalsaha](https://github.com/tamalsaha))
- Support migration from TPR to CRD [\#94](https://github.com/kubedb/elasticsearch/pull/94) ([aerokite](https://github.com/aerokite))
- Use kutil in e2e-test [\#93](https://github.com/kubedb/elasticsearch/pull/93) ([aerokite](https://github.com/aerokite))
- Resume DormantDatabase while creating Original DB again [\#92](https://github.com/kubedb/elasticsearch/pull/92) ([aerokite](https://github.com/aerokite))
- Rewrite e2e tests using ginkgo [\#91](https://github.com/kubedb/elasticsearch/pull/91) ([aerokite](https://github.com/aerokite))

## [0.6.0](https://github.com/kubedb/elasticsearch/tree/0.6.0) (2017-07-24)
[Full Changelog](https://github.com/kubedb/elasticsearch/compare/0.5.0...0.6.0)

**Merged pull requests:**

- Revendor for api fix [\#90](https://github.com/kubedb/elasticsearch/pull/90) ([aerokite](https://github.com/aerokite))

## [0.5.0](https://github.com/kubedb/elasticsearch/tree/0.5.0) (2017-07-19)
[Full Changelog](https://github.com/kubedb/elasticsearch/compare/0.4.0...0.5.0)

## [0.4.0](https://github.com/kubedb/elasticsearch/tree/0.4.0) (2017-07-18)
[Full Changelog](https://github.com/kubedb/elasticsearch/compare/0.3.1...0.4.0)

## [0.3.1](https://github.com/kubedb/elasticsearch/tree/0.3.1) (2017-07-14)
[Full Changelog](https://github.com/kubedb/elasticsearch/compare/0.3.0...0.3.1)

## [0.3.0](https://github.com/kubedb/elasticsearch/tree/0.3.0) (2017-07-08)
[Full Changelog](https://github.com/kubedb/elasticsearch/compare/0.2.0...0.3.0)

**Merged pull requests:**

- 	Support RBAC [\#89](https://github.com/kubedb/elasticsearch/pull/89) ([aerokite](https://github.com/aerokite))
- Use snapshot path prefix  [\#88](https://github.com/kubedb/elasticsearch/pull/88) ([tamalsaha](https://github.com/tamalsaha))
- Allow setting resources for StatefulSet or Snapshot/Restore jobs [\#87](https://github.com/kubedb/elasticsearch/pull/87) ([tamalsaha](https://github.com/tamalsaha))
- Add app=kubedb label to TPR registration [\#86](https://github.com/kubedb/elasticsearch/pull/86) ([tamalsaha](https://github.com/tamalsaha))
- Support non-default service account with offshoot pods [\#85](https://github.com/kubedb/elasticsearch/pull/85) ([tamalsaha](https://github.com/tamalsaha))
- Separate validation [\#84](https://github.com/kubedb/elasticsearch/pull/84) ([aerokite](https://github.com/aerokite))

## [0.2.0](https://github.com/kubedb/elasticsearch/tree/0.2.0) (2017-06-22)
[Full Changelog](https://github.com/kubedb/elasticsearch/compare/0.1.0...0.2.0)

**Merged pull requests:**

- Expose exporter port via service [\#83](https://github.com/kubedb/elasticsearch/pull/83) ([tamalsaha](https://github.com/tamalsaha))
- get summary report [\#82](https://github.com/kubedb/elasticsearch/pull/82) ([aerokite](https://github.com/aerokite))
- Use side-car exporter [\#81](https://github.com/kubedb/elasticsearch/pull/81) ([tamalsaha](https://github.com/tamalsaha))
- Use client-go [\#80](https://github.com/kubedb/elasticsearch/pull/80) ([tamalsaha](https://github.com/tamalsaha))

## [0.1.0](https://github.com/kubedb/elasticsearch/tree/0.1.0) (2017-06-14)
**Fixed bugs:**

- Allow updating to create missing workloads [\#76](https://github.com/kubedb/elasticsearch/pull/76) ([aerokite](https://github.com/aerokite))

**Merged pull requests:**

- Change api version to v1alpha1 [\#79](https://github.com/kubedb/elasticsearch/pull/79) ([tamalsaha](https://github.com/tamalsaha))
- Pass cronController as parameter [\#78](https://github.com/kubedb/elasticsearch/pull/78) ([aerokite](https://github.com/aerokite))
- Use built-in exporter [\#77](https://github.com/kubedb/elasticsearch/pull/77) ([tamalsaha](https://github.com/tamalsaha))
- Add analytics event for operator [\#75](https://github.com/kubedb/elasticsearch/pull/75) ([aerokite](https://github.com/aerokite))
- Add analytics [\#74](https://github.com/kubedb/elasticsearch/pull/74) ([aerokite](https://github.com/aerokite))
- Revendor client-go [\#73](https://github.com/kubedb/elasticsearch/pull/73) ([tamalsaha](https://github.com/tamalsaha))
- Add Run\(\) method to just run controller. [\#72](https://github.com/kubedb/elasticsearch/pull/72) ([tamalsaha](https://github.com/tamalsaha))
- Add HTTP server to expose metrics [\#71](https://github.com/kubedb/elasticsearch/pull/71) ([tamalsaha](https://github.com/tamalsaha))
- Prometheus support [\#70](https://github.com/kubedb/elasticsearch/pull/70) ([saumanbiswas](https://github.com/saumanbiswas))
- Use kubedb docker hub account [\#69](https://github.com/kubedb/elasticsearch/pull/69) ([tamalsaha](https://github.com/tamalsaha))
- Use kubedb instead of k8sdb [\#68](https://github.com/kubedb/elasticsearch/pull/68) ([tamalsaha](https://github.com/tamalsaha))
- Do not handle DormantDatabase [\#67](https://github.com/kubedb/elasticsearch/pull/67) ([aerokite](https://github.com/aerokite))
- Pass clients instead of config [\#66](https://github.com/kubedb/elasticsearch/pull/66) ([aerokite](https://github.com/aerokite))
- Ungroup imports on fmt [\#64](https://github.com/kubedb/elasticsearch/pull/64) ([tamalsaha](https://github.com/tamalsaha))
- Fix go report card issue [\#63](https://github.com/kubedb/elasticsearch/pull/63) ([tamalsaha](https://github.com/tamalsaha))
- Rename DeletedDatabase to DormantDatabase [\#62](https://github.com/kubedb/elasticsearch/pull/62) ([tamalsaha](https://github.com/tamalsaha))
- Add e2e test for updating scheduler [\#61](https://github.com/kubedb/elasticsearch/pull/61) ([aerokite](https://github.com/aerokite))
- Fix update operation [\#60](https://github.com/kubedb/elasticsearch/pull/60) ([aerokite](https://github.com/aerokite))
- Remove prefix from snapshot job [\#59](https://github.com/kubedb/elasticsearch/pull/59) ([aerokite](https://github.com/aerokite))
- Rename DatabaseSnapshot to Snapshot [\#58](https://github.com/kubedb/elasticsearch/pull/58) ([tamalsaha](https://github.com/tamalsaha))
- Modify StatefulSet naming format [\#56](https://github.com/kubedb/elasticsearch/pull/56) ([aerokite](https://github.com/aerokite))
- Get object each time before updating [\#55](https://github.com/kubedb/elasticsearch/pull/55) ([aerokite](https://github.com/aerokite))
- Check docker image version [\#54](https://github.com/kubedb/elasticsearch/pull/54) ([aerokite](https://github.com/aerokite))
- Create headless service for StatefulSet [\#53](https://github.com/kubedb/elasticsearch/pull/53) ([aerokite](https://github.com/aerokite))
- Use data as Volume name [\#52](https://github.com/kubedb/elasticsearch/pull/52) ([aerokite](https://github.com/aerokite))
- Use kind in label instead of type [\#50](https://github.com/kubedb/elasticsearch/pull/50) ([aerokite](https://github.com/aerokite))
- Do not store autogenerated meta information [\#49](https://github.com/kubedb/elasticsearch/pull/49) ([aerokite](https://github.com/aerokite))
- Bubble up error for controller methods [\#47](https://github.com/kubedb/elasticsearch/pull/47) ([aerokite](https://github.com/aerokite))
- Modify e2e test. Do not support recovery by recreating Elastic anymore. [\#46](https://github.com/kubedb/elasticsearch/pull/46) ([aerokite](https://github.com/aerokite))
- Use Kubernetes EventRecorder directly [\#45](https://github.com/kubedb/elasticsearch/pull/45) ([aerokite](https://github.com/aerokite))
- Address status field changes [\#44](https://github.com/kubedb/elasticsearch/pull/44) ([aerokite](https://github.com/aerokite))
- Use canary tag for k8sdb images [\#42](https://github.com/kubedb/elasticsearch/pull/42) ([aerokite](https://github.com/aerokite))
- Install ca-certificates in operator docker image. [\#41](https://github.com/kubedb/elasticsearch/pull/41) ([tamalsaha](https://github.com/tamalsaha))
- Add deployment.yaml [\#40](https://github.com/kubedb/elasticsearch/pull/40) ([aerokite](https://github.com/aerokite))
- Rename "destroy" to "wipeOut" [\#38](https://github.com/kubedb/elasticsearch/pull/38) ([tamalsaha](https://github.com/tamalsaha))
- Store Elastic Spec in DeletedDatabase [\#36](https://github.com/kubedb/elasticsearch/pull/36) ([aerokite](https://github.com/aerokite))
- Update timing fields. [\#35](https://github.com/kubedb/elasticsearch/pull/35) ([tamalsaha](https://github.com/tamalsaha))
- Use k8sdb docker hub account [\#34](https://github.com/kubedb/elasticsearch/pull/34) ([tamalsaha](https://github.com/tamalsaha))
- Implement database initialization [\#32](https://github.com/kubedb/elasticsearch/pull/32) ([aerokite](https://github.com/aerokite))
- Use resource name constant from apimachinery [\#31](https://github.com/kubedb/elasticsearch/pull/31) ([tamalsaha](https://github.com/tamalsaha))
- Use one controller struct [\#30](https://github.com/kubedb/elasticsearch/pull/30) ([tamalsaha](https://github.com/tamalsaha))
- Implement updated interfaces. [\#29](https://github.com/kubedb/elasticsearch/pull/29) ([tamalsaha](https://github.com/tamalsaha))
- Rename controller image to k8s-es [\#28](https://github.com/kubedb/elasticsearch/pull/28) ([tamalsaha](https://github.com/tamalsaha))
- Implement Snapshotter, Deleter with Controller [\#27](https://github.com/kubedb/elasticsearch/pull/27) ([aerokite](https://github.com/aerokite))
- Modify implementation [\#26](https://github.com/kubedb/elasticsearch/pull/26) ([aerokite](https://github.com/aerokite))
- Implement interface [\#25](https://github.com/kubedb/elasticsearch/pull/25) ([aerokite](https://github.com/aerokite))
- Reorganize code [\#24](https://github.com/kubedb/elasticsearch/pull/24) ([aerokite](https://github.com/aerokite))
- Modify snapshot name format [\#23](https://github.com/kubedb/elasticsearch/pull/23) ([aerokite](https://github.com/aerokite))
- Modify controller for backup operation [\#22](https://github.com/kubedb/elasticsearch/pull/22) ([aerokite](https://github.com/aerokite))
- Use osm to pull/push snapshots [\#21](https://github.com/kubedb/elasticsearch/pull/21) ([aerokite](https://github.com/aerokite))
- Move api & client to apimachinery [\#20](https://github.com/kubedb/elasticsearch/pull/20) ([aerokite](https://github.com/aerokite))
- Remove DeleteOptions{} [\#18](https://github.com/kubedb/elasticsearch/pull/18) ([aerokite](https://github.com/aerokite))
- Modify labels and annotations [\#17](https://github.com/kubedb/elasticsearch/pull/17) ([aerokite](https://github.com/aerokite))
- Add controller operation [\#16](https://github.com/kubedb/elasticsearch/pull/16) ([aerokite](https://github.com/aerokite))
- Modify types to match TPR "elastic.k8sdb.com" [\#14](https://github.com/kubedb/elasticsearch/pull/14) ([aerokite](https://github.com/aerokite))
- Change Kind "elasticsearch" to "elastic" [\#13](https://github.com/kubedb/elasticsearch/pull/13) ([aerokite](https://github.com/aerokite))
- Move elasticsearch\_discovery & docker files [\#6](https://github.com/kubedb/elasticsearch/pull/6) ([aerokite](https://github.com/aerokite))
- Modify skeleton to elasticsearch [\#4](https://github.com/kubedb/elasticsearch/pull/4) ([aerokite](https://github.com/aerokite))



\* *This Change Log was automatically generated by [github_changelog_generator](https://github.com/skywinder/Github-Changelog-Generator)*