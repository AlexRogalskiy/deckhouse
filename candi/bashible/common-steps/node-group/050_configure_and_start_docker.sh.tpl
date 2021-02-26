{{- if eq .cri "Docker" }}

bb-event-on 'bb-sync-file-changed' '_on_docker_config_changed'
_on_docker_config_changed() {
{{ if ne .runType "ImageBuilding" -}}
  bb-deckhouse-get-disruptive-update-approval
  systemctl restart docker.service
{{- end }}
}

mkdir -p /etc/docker
bb-sync-file /etc/docker/daemon.json - << "EOF"
{
{{- $max_concurrent_downloads := 3 }}
{{- if hasKey .nodeGroup.cri "docker" }}
  {{- $max_concurrent_downloads = .nodeGroup.cri.docker.maxConcurrentDownloads | default $max_concurrent_downloads }}
{{- end }}
        "log-driver": "json-file",
        "log-opts": {
                "max-file": "5",
                "max-size": "10m"
        },
	"max-concurrent-downloads": {{ $max_concurrent_downloads }}
}
EOF
{{- end }}
