package deckhouse

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/iancoleman/strcase"
	appsv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/deckhouse/deckhouse/candictl/pkg/config"
	"github.com/deckhouse/deckhouse/candictl/pkg/kubernetes/actions"
	"github.com/deckhouse/deckhouse/candictl/pkg/kubernetes/actions/manifests"
	"github.com/deckhouse/deckhouse/candictl/pkg/kubernetes/client"
	"github.com/deckhouse/deckhouse/candictl/pkg/log"
	"github.com/deckhouse/deckhouse/candictl/pkg/util/retry"
)

type Config struct {
	Registry              string
	DockerCfg             string
	LogLevel              string
	Bundle                string
	ReleaseChannel        string
	DevBranch             string
	UUID                  string
	KubeDNSAddress        string
	ClusterConfig         []byte
	ProviderClusterConfig []byte
	StaticClusterConfig   []byte
	TerraformState        []byte
	NodesTerraformState   map[string][]byte
	CloudDiscovery        []byte
	DeckhouseConfig       map[string]interface{}
}

func (c *Config) GetImage() string {
	registryNameTemplate := "%s/dev:%s"
	tag := c.DevBranch
	if c.ReleaseChannel != "" {
		registryNameTemplate = "%s:%s"
		tag = strcase.ToKebab(c.ReleaseChannel)
	}
	return fmt.Sprintf(registryNameTemplate, c.Registry, tag)
}

func (c *Config) IsRegistryAccessRequired() bool {
	return c.DockerCfg != ""
}

func deckhouseDeploymentFromConfig(cfg *Config) *appsv1.Deployment {
	return manifests.DeckhouseDeployment(cfg.GetImage(), cfg.LogLevel, cfg.Bundle, cfg.IsRegistryAccessRequired())
}

func CreateDeckhouseManifests(kubeCl *client.KubernetesClient, cfg *Config) error {
	tasks := []actions.ManifestTask{
		{
			Name:     `Namespace "d8-system"`,
			Manifest: func() interface{} { return manifests.DeckhouseNamespace("d8-system") },
			CreateFunc: func(manifest interface{}) error {
				_, err := kubeCl.CoreV1().Namespaces().Create(manifest.(*apiv1.Namespace))
				return err
			},
			UpdateFunc: func(manifest interface{}) error {
				_, err := kubeCl.CoreV1().Namespaces().Update(manifest.(*apiv1.Namespace))
				return err
			},
		},
		{
			Name:     `Admin ClusterRole "cluster-admin"`,
			Manifest: func() interface{} { return manifests.DeckhouseAdminClusterRole() },
			CreateFunc: func(manifest interface{}) error {
				_, err := kubeCl.RbacV1().ClusterRoles().Create(manifest.(*rbacv1.ClusterRole))
				return err
			},
			UpdateFunc: func(manifest interface{}) error {
				_, err := kubeCl.RbacV1().ClusterRoles().Update(manifest.(*rbacv1.ClusterRole))
				return err
			},
		},
		{
			Name:     `ClusterRoleBinding "deckhouse"`,
			Manifest: func() interface{} { return manifests.DeckhouseAdminClusterRoleBinding() },
			CreateFunc: func(manifest interface{}) error {
				_, err := kubeCl.RbacV1().ClusterRoleBindings().Create(manifest.(*rbacv1.ClusterRoleBinding))
				return err
			},
			UpdateFunc: func(manifest interface{}) error {
				_, err := kubeCl.RbacV1().ClusterRoleBindings().Update(manifest.(*rbacv1.ClusterRoleBinding))
				return err
			},
		},
		{
			Name:     `ServiceAccount "deckhouse"`,
			Manifest: func() interface{} { return manifests.DeckhouseServiceAccount() },
			CreateFunc: func(manifest interface{}) error {
				_, err := kubeCl.CoreV1().ServiceAccounts("d8-system").Create(manifest.(*apiv1.ServiceAccount))
				return err
			},
			UpdateFunc: func(manifest interface{}) error {
				_, err := kubeCl.CoreV1().ServiceAccounts("d8-system").Update(manifest.(*apiv1.ServiceAccount))
				return err
			},
		},
		{
			Name:     `ConfigMap "deckhouse"`,
			Manifest: func() interface{} { return manifests.DeckhouseConfigMap(cfg.DeckhouseConfig) },
			CreateFunc: func(manifest interface{}) error {
				_, err := kubeCl.CoreV1().ConfigMaps("d8-system").Create(manifest.(*apiv1.ConfigMap))
				return err
			},
			UpdateFunc: func(manifest interface{}) error {
				_, err := kubeCl.CoreV1().ConfigMaps("d8-system").Update(manifest.(*apiv1.ConfigMap))
				return err
			},
		},
	}

	if cfg.IsRegistryAccessRequired() {
		tasks = append(tasks, actions.ManifestTask{
			Name:     `Secret "deckhouse-registry"`,
			Manifest: func() interface{} { return manifests.DeckhouseRegistrySecret(cfg.DockerCfg) },
			CreateFunc: func(manifest interface{}) error {
				_, err := kubeCl.CoreV1().Secrets("d8-system").Create(manifest.(*apiv1.Secret))
				return err
			},
			UpdateFunc: func(manifest interface{}) error {
				_, err := kubeCl.CoreV1().Secrets("d8-system").Update(manifest.(*apiv1.Secret))
				return err
			},
		})
	}

	if len(cfg.TerraformState) > 0 {
		tasks = append(tasks, actions.ManifestTask{
			Name:     `Secret "d8-cluster-terraform-state"`,
			Manifest: func() interface{} { return manifests.SecretWithTerraformState(cfg.TerraformState) },
			CreateFunc: func(manifest interface{}) error {
				_, err := kubeCl.CoreV1().Secrets("d8-system").Create(manifest.(*apiv1.Secret))
				return err
			},
			UpdateFunc: func(manifest interface{}) error {
				_, err := kubeCl.CoreV1().Secrets("d8-system").Update(manifest.(*apiv1.Secret))
				return err
			},
		})
	}

	for nodeName, tfState := range cfg.NodesTerraformState {
		getManifest := func() interface{} { return manifests.SecretWithNodeTerraformState(nodeName, "master", tfState, nil) }
		tasks = append(tasks, actions.ManifestTask{
			Name:     fmt.Sprintf(`Secret "d8-node-terraform-state-%s"`, nodeName),
			Manifest: getManifest,
			CreateFunc: func(manifest interface{}) error {
				_, err := kubeCl.CoreV1().Secrets("d8-system").Create(manifest.(*apiv1.Secret))
				return err
			},
			UpdateFunc: func(manifest interface{}) error {
				_, err := kubeCl.CoreV1().Secrets("d8-system").Update(manifest.(*apiv1.Secret))
				return err
			},
		})
	}

	if len(cfg.ClusterConfig) > 0 {
		tasks = append(tasks, actions.ManifestTask{
			Name:     `Secret "d8-cluster-configuration"`,
			Manifest: func() interface{} { return manifests.SecretWithClusterConfig(cfg.ClusterConfig) },
			CreateFunc: func(manifest interface{}) error {
				_, err := kubeCl.CoreV1().Secrets("kube-system").Create(manifest.(*apiv1.Secret))
				return err
			},
			UpdateFunc: func(manifest interface{}) error {
				_, err := kubeCl.CoreV1().Secrets("kube-system").Update(manifest.(*apiv1.Secret))
				return err
			},
		})
	}

	if len(cfg.ProviderClusterConfig) > 0 {
		tasks = append(tasks, actions.ManifestTask{
			Name: `Secret "d8-provider-cluster-configuration"`,
			Manifest: func() interface{} {
				return manifests.SecretWithProviderClusterConfig(
					cfg.ProviderClusterConfig, cfg.CloudDiscovery,
				)
			},
			CreateFunc: func(manifest interface{}) error {
				_, err := kubeCl.CoreV1().Secrets("kube-system").Create(manifest.(*apiv1.Secret))
				return err
			},
			UpdateFunc: func(manifest interface{}) error {
				data, err := json.Marshal(manifest.(*apiv1.Secret))
				if err != nil {
					return err
				}
				_, err = kubeCl.CoreV1().Secrets("kube-system").Patch(
					"d8-provider-cluster-configuration",
					types.MergePatchType,
					data,
				)
				return err
			},
		})
	}

	if len(cfg.StaticClusterConfig) > 0 {
		tasks = append(tasks, actions.ManifestTask{
			Name: `Secret "d8-static-cluster-configuration"`,
			Manifest: func() interface{} {
				return manifests.SecretWithStaticClusterConfig(cfg.StaticClusterConfig)
			},
			CreateFunc: func(manifest interface{}) error {
				_, err := kubeCl.CoreV1().Secrets("kube-system").Create(manifest.(*apiv1.Secret))
				return err
			},
			UpdateFunc: func(manifest interface{}) error {
				data, err := json.Marshal(manifest.(*apiv1.Secret))
				if err != nil {
					return err
				}
				_, err = kubeCl.CoreV1().Secrets("kube-system").Patch(
					"d8-static-cluster-configuration",
					types.MergePatchType,
					data,
				)
				return err
			},
		})
	}

	if len(cfg.UUID) > 0 {
		tasks = append(tasks, actions.ManifestTask{
			Name: `ConfigMap "d8-cluster-uuid"`,
			Manifest: func() interface{} {
				return manifests.ClusterUUIDConfigMap(cfg.UUID)
			},
			CreateFunc: func(manifest interface{}) error {
				_, err := kubeCl.CoreV1().ConfigMaps("kube-system").Create(manifest.(*apiv1.ConfigMap))
				return err
			},
			UpdateFunc: func(manifest interface{}) error {
				_, err := kubeCl.CoreV1().ConfigMaps("kube-system").Update(manifest.(*apiv1.ConfigMap))
				return err
			},
		})
	}

	if cfg.KubeDNSAddress != "" {
		tasks = append(tasks, actions.ManifestTask{
			Name: `Service "kube-dns"`,
			Manifest: func() interface{} {
				return manifests.KubeDNSService(cfg.KubeDNSAddress)
			},
			CreateFunc: func(manifest interface{}) error {
				_, err := kubeCl.CoreV1().Services("kube-system").Create(manifest.(*apiv1.Service))
				if err != nil && strings.Contains(err.Error(), "provided IP is already allocated") {
					log.InfoLn("Service for DNS already exists. Skip!")
					return nil
				}
				return err
			},
			UpdateFunc: func(manifest interface{}) error {
				_, err := kubeCl.CoreV1().Services("kube-system").Update(manifest.(*apiv1.Service))
				return err
			},
		})
	}

	tasks = append(tasks, actions.ManifestTask{
		Name: `Deployment "deckhouse"`,
		Manifest: func() interface{} {
			return deckhouseDeploymentFromConfig(cfg)
		},
		CreateFunc: func(manifest interface{}) error {
			_, err := kubeCl.AppsV1().Deployments("d8-system").Create(manifest.(*appsv1.Deployment))
			return err
		},
		UpdateFunc: func(manifest interface{}) error {
			_, err := kubeCl.AppsV1().Deployments("d8-system").Update(manifest.(*appsv1.Deployment))
			return err
		},
	})

	return log.Process("default", "Create Manifests", func() error {
		for _, task := range tasks {
			err := task.CreateOrUpdate()
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func WaitForReadiness(kubeCl *client.KubernetesClient) error {
	return log.Process("default", "Waiting for Deckhouse to become Ready", func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()
		for {
			select {
			case <-ctx.Done():
				return ErrTimedOut
			default:
				ok, err := NewLogPrinter(kubeCl).WaitPodBecomeReady().Print(ctx)
				if err != nil {
					if errors.Is(err, ErrTimedOut) {
						return err
					}
					log.InfoLn(err.Error())
				}

				if ok {
					log.InfoLn("Deckhouse pod is Ready!")
					return nil
				}

				time.Sleep(5 * time.Second)
			}
		}
	})
}

func CreateDeckhouseDeployment(kubeCl *client.KubernetesClient, cfg *Config) error {
	task := actions.ManifestTask{
		Name: `Deployment "deckhouse"`,
		Manifest: func() interface{} {
			return manifests.DeckhouseDeployment(cfg.GetImage(), cfg.LogLevel, cfg.Bundle, cfg.IsRegistryAccessRequired())
		},
		CreateFunc: func(manifest interface{}) error {
			_, err := kubeCl.AppsV1().Deployments("d8-system").Create(manifest.(*appsv1.Deployment))
			return err
		},
		UpdateFunc: func(manifest interface{}) error {
			_, err := kubeCl.AppsV1().Deployments("d8-system").Update(manifest.(*appsv1.Deployment))
			return err
		},
	}

	return log.Process("default", "Create Deployment", task.CreateOrUpdate)
}

func CreateDeckhouseDeploymentManifest(cfg *Config) *appsv1.Deployment {
	return manifests.DeckhouseDeployment(cfg.GetImage(), cfg.LogLevel, cfg.Bundle, cfg.IsRegistryAccessRequired())
}

func WaitForKubernetesAPI(kubeCl *client.KubernetesClient) error {
	return retry.StartLoop("Waiting for Kubernetes API to become Ready", 45, 5, func() error {
		_, err := kubeCl.Discovery().ServerVersion()
		if err == nil {
			return nil
		}
		return fmt.Errorf("kubernetes API is not Ready: %w", err)
	})
}

func PrepareDeckhouseInstallConfig(metaConfig *config.MetaConfig) (*Config, error) {
	clusterConfig, err := metaConfig.ClusterConfigYAML()
	if err != nil {
		return nil, fmt.Errorf("marshal cluster config: %v", err)
	}

	providerClusterConfig, err := metaConfig.ProviderClusterConfigYAML()
	if err != nil {
		return nil, fmt.Errorf("marshal provider config: %v", err)
	}

	staticClusterConfig, err := metaConfig.StaticClusterConfigYAML()
	if err != nil {
		return nil, fmt.Errorf("marshal static config: %v", err)
	}

	installConfig := Config{
		UUID:                  metaConfig.UUID,
		Registry:              metaConfig.DeckhouseConfig.ImagesRepo,
		DockerCfg:             metaConfig.DeckhouseConfig.RegistryDockerCfg,
		DevBranch:             metaConfig.DeckhouseConfig.DevBranch,
		ReleaseChannel:        metaConfig.DeckhouseConfig.ReleaseChannel,
		Bundle:                metaConfig.DeckhouseConfig.Bundle,
		LogLevel:              metaConfig.DeckhouseConfig.LogLevel,
		DeckhouseConfig:       metaConfig.MergeDeckhouseConfig(),
		KubeDNSAddress:        metaConfig.ClusterDNSAddress,
		ProviderClusterConfig: providerClusterConfig,
		StaticClusterConfig:   staticClusterConfig,
		ClusterConfig:         clusterConfig,
	}

	return &installConfig, nil
}
