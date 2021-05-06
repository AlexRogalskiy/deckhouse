package deckhouse

import (
	"fmt"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/deckhouse/deckhouse/dhctl/pkg/kubernetes/client"
	"github.com/deckhouse/deckhouse/dhctl/pkg/log"
)

func TestDeckhouseInstall(t *testing.T) {
	log.InitLogger("simple")
	fakeClient := client.NewFakeKubernetesClient()

	tests := []struct {
		name    string
		test    func() error
		wantErr bool
	}{
		{
			"Empty config",
			func() error {
				return CreateDeckhouseManifests(fakeClient, &Config{})
			},
			false,
		},
		{
			"Double install",
			func() error {
				err := CreateDeckhouseManifests(fakeClient, &Config{})
				if err != nil {
					return err
				}
				return CreateDeckhouseManifests(fakeClient, &Config{})
			},
			false,
		},
		{
			"With docker cfg",
			func() error {
				err := CreateDeckhouseManifests(fakeClient, &Config{DockerCfg: "YW55dGhpbmc="})
				if err != nil {
					return err
				}
				s, err := fakeClient.CoreV1().Secrets("d8-system").Get("deckhouse-registry", metav1.GetOptions{})
				if err != nil {
					return err
				}

				dockercfg := s.Data[".dockerconfigjson"]
				if string(dockercfg) != "anything" {
					return fmt.Errorf(".dockercfg data: %s", dockercfg)
				}
				return nil
			},
			false,
		},
		{
			"With secrets",
			func() error {
				config := Config{
					ClusterConfig:         []byte(`test`),
					ProviderClusterConfig: []byte(`test`),
					TerraformState:        []byte(`test`),
					DeckhouseConfig:       map[string]interface{}{"test": "test"},
				}
				err := CreateDeckhouseManifests(fakeClient, &config)
				if err != nil {
					return err
				}
				return nil
			},
			false,
		},
	}

	for _, tc := range tests {
		err := tc.test()

		if err != nil && !tc.wantErr {
			t.Errorf("%s: %v", tc.name, err)
		}

		if err == nil && tc.wantErr {
			t.Errorf("%s: expected error, didn't get one", tc.name)
		}
	}
}
