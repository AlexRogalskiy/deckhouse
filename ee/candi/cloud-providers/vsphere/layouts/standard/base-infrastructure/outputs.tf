# Copyright 2021 Flant JSC
# Licensed under the Deckhouse Platform Enterprise Edition (EE) license. See https://github.com/deckhouse/deckhouse/blob/main/ee/LICENSE

output "cloud_discovery_data" {
  value = {
    "apiVersion" = "deckhouse.io/v1"
    "kind" = "VsphereCloudDiscoveryData"
    "vmFolderPath" = vsphere_folder.main.path
    "resourcePoolPath" = local.base_resource_pool != "" ? join("/", [local.base_resource_pool, local.prefix]) : local.prefix
  }
}
