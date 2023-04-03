
<a name="v0.0.6"></a>
# [v0.0.6] - 2023-04-03
### Bug Fixes 🐞
- shorten custom data in cloud init files ([#121](https://github.com/Azure/aks-engine-azurestack/issues/121))
- add kube-addon-manager v9.1.6 to vhd ([#118](https://github.com/Azure/aks-engine-azurestack/issues/118))
- remove invalid k8s v1.24 flags ([#114](https://github.com/Azure/aks-engine-azurestack/issues/114))
- use cross-platform pause image as the containerd sandbox image on Windows ([#106](https://github.com/Azure/aks-engine-azurestack/issues/106))
- kubernetes-azurestack.json uses distro aks-ubuntu-20.04 ([#87](https://github.com/Azure/aks-engine-azurestack/issues/87))
- CoreDNS image not updated after cluster upgrade ([#75](https://github.com/Azure/aks-engine-azurestack/issues/75))
- change reference of cni config to scripts dir ([#71](https://github.com/Azure/aks-engine-azurestack/issues/71))
- add Azure CNI config script to Ubuntu VHD ([#70](https://github.com/Azure/aks-engine-azurestack/issues/70))
- unit test checking e2e configs ([#49](https://github.com/Azure/aks-engine-azurestack/issues/49))
- syntax error in Windows VHD script ([#41](https://github.com/Azure/aks-engine-azurestack/issues/41))
- ensure eth0 addr is set to NIC's primary addr ([#39](https://github.com/Azure/aks-engine-azurestack/issues/39))

### Code Style 🎶
- replace usage of deprecated "io/ioutil" golang package

### Continuous Integration 💜
- update release Github action to test k8s v1.24 & v1.25
- exclude version control information from test binary ([#122](https://github.com/Azure/aks-engine-azurestack/issues/122))
- Add -buildvcs=false for go build ([#120](https://github.com/Azure/aks-engine-azurestack/issues/120))
- call test-vhd-no-egress github workflow from create-release-branch ([#119](https://github.com/Azure/aks-engine-azurestack/issues/119))
- add no-egress GitHub action ([#108](https://github.com/Azure/aks-engine-azurestack/issues/108))
- release workflow tags the correct commit ([#113](https://github.com/Azure/aks-engine-azurestack/issues/113))
- gen-release-changelog wf creates branch and commit ([#112](https://github.com/Azure/aks-engine-azurestack/issues/112))
- update actions/checkout to v3 ([#111](https://github.com/Azure/aks-engine-azurestack/issues/111))
- chocolatey workflow ([#86](https://github.com/Azure/aks-engine-azurestack/issues/86))
- release workflows run no-egress scenarios ([#85](https://github.com/Azure/aks-engine-azurestack/issues/85))
- remove no-egress job from create branch action ([#79](https://github.com/Azure/aks-engine-azurestack/issues/79))
- PR gate runs E2E suite ([#69](https://github.com/Azure/aks-engine-azurestack/issues/69))
- PR checks consume SIG images ([#64](https://github.com/Azure/aks-engine-azurestack/issues/64))
- E2E PR check uses user assigned identity ([#54](https://github.com/Azure/aks-engine-azurestack/issues/54))
- fix variable name in e2e PR check ([#52](https://github.com/Azure/aks-engine-azurestack/issues/52))
- e2e PR check sets tenant ([#51](https://github.com/Azure/aks-engine-azurestack/issues/51))
- e2e PR check does not use AvailabilitySets ([#47](https://github.com/Azure/aks-engine-azurestack/issues/47))
- e2e PR check does not use custom VNET ([#46](https://github.com/Azure/aks-engine-azurestack/issues/46))
- e2e PR check does not use MSI ([#45](https://github.com/Azure/aks-engine-azurestack/issues/45))

### Documentation 📘
- Enable azurestack-csi-driver addon for mixed clusters ([#96](https://github.com/Azure/aks-engine-azurestack/issues/96))
- remove Azure as a target cloud ([#43](https://github.com/Azure/aks-engine-azurestack/issues/43))
- rename binary name in all markdown files ([#42](https://github.com/Azure/aks-engine-azurestack/issues/42))

### Features 🌈
- add support for Kubernetes v1.24.11 ([#109](https://github.com/Azure/aks-engine-azurestack/issues/109))
- migrate from Pod Security Policy to Pod Security admission ([#94](https://github.com/Azure/aks-engine-azurestack/issues/94))
- DISA Ubuntu 20.04 STIG compliance ([#83](https://github.com/Azure/aks-engine-azurestack/issues/83))

### Maintenance 🔧
- update Linux and Windows VHDs for March 2023 ([#115](https://github.com/Azure/aks-engine-azurestack/issues/115))
- support Kubernetes v1.25.7 ([#105](https://github.com/Azure/aks-engine-azurestack/issues/105))
- upgrade coredns to v1.9.4 ([#98](https://github.com/Azure/aks-engine-azurestack/issues/98))
- upgrade containerd to 1.5.16 ([#95](https://github.com/Azure/aks-engine-azurestack/issues/95))
- upgrade pause to v3.8 ([#93](https://github.com/Azure/aks-engine-azurestack/issues/93))
- update golang toolchain to v1.19 ([#90](https://github.com/Azure/aks-engine-azurestack/issues/90))
- update registries for nvidia and k8s.io components ([#88](https://github.com/Azure/aks-engine-azurestack/issues/88))
- remove package apache2-utils from VHD ([#82](https://github.com/Azure/aks-engine-azurestack/issues/82))
- update default windows image to jan 2023 ([#77](https://github.com/Azure/aks-engine-azurestack/issues/77))
- Update Windows VHD packer job to use Jan 2023 patches ([#76](https://github.com/Azure/aks-engine-azurestack/issues/76))
- change base image sku and version to azurestack ([#74](https://github.com/Azure/aks-engine-azurestack/issues/74))
- set fsType to ext4 in supported storage classes ([#73](https://github.com/Azure/aks-engine-azurestack/issues/73))
- enable v1.23.15 & v1.24.9, use ubuntu 20.04 as default, force containerd runtime ([#68](https://github.com/Azure/aks-engine-azurestack/issues/68))
- include relevant updates from v0.75.0 ([#56](https://github.com/Azure/aks-engine-azurestack/issues/56))
- include relevant updates from v0.74.0 ([#55](https://github.com/Azure/aks-engine-azurestack/issues/55))
- include relevant updates from v0.73.0 ([#53](https://github.com/Azure/aks-engine-azurestack/issues/53))
- remove kv-fluxvolume addon ([#48](https://github.com/Azure/aks-engine-azurestack/issues/48))
- prefer ADO for PR E2E check ([#38](https://github.com/Azure/aks-engine-azurestack/issues/38))
- added e2e to PR workflow ([#36](https://github.com/Azure/aks-engine-azurestack/issues/36))

### Security Fix 🛡️
- bump x/net and x/crypto ([#104](https://github.com/Azure/aks-engine-azurestack/issues/104))

### Testing 💚
- remove auditd from packages validate script ([#116](https://github.com/Azure/aks-engine-azurestack/issues/116))
- e2e suite validates an existing PV works after a cluster upgrade ([#92](https://github.com/Azure/aks-engine-azurestack/issues/92))
- e2e sets ImageRef in all linux nodepools ([#65](https://github.com/Azure/aks-engine-azurestack/issues/65))

#### Please report any issues here: https://github.com/Azure/aks-engine-azurestack/issues/new
[Unreleased]: https://github.com/Azure/aks-engine-azurestack/compare/v0.0.6...HEAD
[v0.0.6]: https://github.com/Azure/aks-engine-azurestack/compare/v0.71.1...v0.0.6
