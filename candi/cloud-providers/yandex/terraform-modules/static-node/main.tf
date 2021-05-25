data "yandex_vpc_subnet" "kube_a" {
  name = "${local.prefix}-a"
}

data "yandex_vpc_subnet" "kube_b" {
  name = "${local.prefix}-b"
}

data "yandex_vpc_subnet" "kube_c" {
  name = "${local.prefix}-c"
}

locals {
  zone_to_subnet = {
    "ru-central1-a" = data.yandex_vpc_subnet.kube_a
    "ru-central1-b" = data.yandex_vpc_subnet.kube_b
    "ru-central1-c" = data.yandex_vpc_subnet.kube_c
  }

  actual_zones = lookup(var.providerClusterConfiguration, "zones", null) != null ? tolist(setintersection(keys(local.zone_to_subnet), var.providerClusterConfiguration.zones)) : keys(local.zone_to_subnet)
  zones = lookup(local.ng, "zones", null) != null ? tolist(setintersection(local.actual_zones, local.ng["zones"])) : local.actual_zones
  subnets = length(local.zones) > 0 ? [for z in local.zones : local.zone_to_subnet[z]] : values(local.zone_to_subnet)
  internal_subnet = element(local.subnets, var.nodeIndex)

  assign_external_ip_address = (local.external_subnet_id == null) && (length(local.external_ip_addresses) > 0) ? true : false
  external_ip = length(local.external_ip_addresses) > 0 ? local.external_ip_addresses[var.nodeIndex] : null
}

resource "yandex_compute_instance" "static" {
  name         = join("-", [local.prefix, var.nodeGroupName, var.nodeIndex])
  hostname     = join("-", [local.prefix, var.nodeGroupName, var.nodeIndex])
  zone         = local.internal_subnet.zone

  allow_stopping_for_update = true

  platform_id  = "standard-v2"
  resources {
    cores  = local.cores
    core_fraction = local.core_fraction
    memory = local.memory
  }

  labels = local.additional_labels

  boot_disk {
    initialize_params {
      type = "network-ssd"
      image_id = local.image_id
      size = local.disk_size_gb
    }
  }

  dynamic "network_interface" {
    for_each = local.external_subnet_id != null ? [local.external_subnet_id] : []
    content {
      subnet_id = network_interface.value
      nat       = false
    }
  }

  network_interface {
    subnet_id = local.internal_subnet.id
    nat       = local.assign_external_ip_address
    nat_ip_address = local.assign_external_ip_address && (local.external_ip != "Auto") ? local.external_ip : null
  }

  network_acceleration_type = local.network_type

  lifecycle {
    ignore_changes = [
      metadata,
      secondary_disk,
    ]
  }

  metadata = {
    ssh-keys = "user:${local.ssh_public_key}"
    user-data = base64decode(var.cloudConfig)
  }
}
