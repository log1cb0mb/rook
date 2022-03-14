# Major Themes

v1.8...

## K8s Version Support

## Upgrade Guides

## Breaking Changes

<<<<<<< HEAD
- Flex driver is fully deprecated. If you are still using flex volumes, before upgrading to v1.8
  you will need to convert them to csi volumes. See the flex conversion tool.
- Min supported version of K8s is now 1.16. If running on an older version of K8s it is recommended
  to update to a newer version before updating to Rook v1.8.
- Directory structure of the YAML examples has changed. Files are now in `deploy/examples` and subdirectories.

### Ceph

=======
* The mds liveness and startup probes are now configured by the filesystem CR instead of the cluster CR. To apply the mds probes, they need to be specified in the filesystem CR. See the [filesystem CR doc](Documentation/ceph-filesystem-crd.md#metadata-server-settings) for more details. See #9550
* In the helm charts, all Ceph components now have default values for the pod resources. The values can be modified or removed in values.yaml depending on cluster requirements.
* Prometheus rules are installed by the helm chart. If you were relying on the cephcluster setting `monitoring.enabled` to create the prometheus rules, they instead need to be enabled by setting `monitoring.createPrometheusRules` in the helm chart values.
* The `region` field for OBC Storage class is ignored, the RGW server always works with s3 client using `us-east-1` as region.
 
>>>>>>> fc2b8012c (object: use us-east-1 for aws go lang sdk)
## Features

- The Rook Operator does not use "tini" as an init process. Instead, it uses the "rook" and handles
  signals on its own.
- Rook adds a finalizer `ceph.rook.io/disaster-protection` to resources critical to the Ceph cluster
  (rook-ceph-mon secrets and configmap) so that the resources will not be accidentally deleted.
- Add support for [Kubernetes Authentication when using HashiCorp Vault Key Management Service](Documentation/ceph-kms.md##kubernetes-based-authentication).
- Bucket notification supported via CRDs
- The failure domain of a pool can be changed on the CephBlockPool instead of requiring toolbox commands
- The Rook Operator and the toolbox now run under the "rook" user and does not use "root" anymore.
- The Operator image now includes the `s5cmd` binary to interact with S3 gateways.
