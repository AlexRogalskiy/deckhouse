# Copyright 2021 Flant JSC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
# policycoreutils-python libseccomp - containerd.io dependencies
SYSTEM_PACKAGES="curl wget virt-what bash-completion lvm2 parted sudo yum-utils yum-plugin-versionlock nfs-utils tar xz device-mapper-persistent-data net-tools policycoreutils-python libseccomp"

KUBERNETES_DEPENDENCIES="conntrack ebtables ethtool iproute iptables socat util-linux"
# yum-plugin-versionlock is needed for bb-yum-install
bb-yum-install yum-plugin-versionlock

bb-yum-install ${SYSTEM_PACKAGES} ${KUBERNETES_DEPENDENCIES}

bb-rp-install "jq:1.6" "bash-completion-extras:2.1-11-centos7" "inotify-tools:3.14-9-centos7" "curl:7.79.1"

bb-yum-remove yum-cron
