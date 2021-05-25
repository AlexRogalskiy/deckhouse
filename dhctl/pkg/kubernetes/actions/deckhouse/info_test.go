package deckhouse

import (
	"context"
	"testing"

	apiv1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"

	"github.com/deckhouse/deckhouse/dhctl/pkg/kubernetes/actions"
	"github.com/deckhouse/deckhouse/dhctl/pkg/kubernetes/actions/manifests"
	"github.com/deckhouse/deckhouse/dhctl/pkg/kubernetes/client"
	"github.com/deckhouse/deckhouse/dhctl/pkg/log"
)

func TestGetClusterInfo(t *testing.T) {
	log.InitLogger("simple")

	tests := []struct {
		name           string
		configMapData  string
		expectedResult string
	}{
		{
			"With proper data",
			`
global:
  clusterName: test
  project: projectTest
`,
			"Cluster:\ttest\nProject:\tprojectTest\n",
		},
		{
			"Without global",
			`
testModule:
  test: test
`,
			"Cluster:\t\nProject:\t\n",
		},
		{
			"Without ConfigMap",
			``,
			"",
		},
	}

	for _, tc := range tests {
		fakeClient := client.NewFakeKubernetesClient()

		var data map[string]interface{}
		err := yaml.Unmarshal([]byte(tc.configMapData), &data)
		if err != nil {
			t.Fatalf("%s: Unexpected error: %v", tc.name, err)
		}

		task := actions.ManifestTask{
			Name: `ConfigMap "deckhouse"`,
			Manifest: func() interface{} {
				if tc.configMapData == "" {
					return &apiv1.ConfigMap{}
				}
				return manifests.DeckhouseConfigMap(data)
			},
			CreateFunc: func(manifest interface{}) error {
				_, err := fakeClient.CoreV1().ConfigMaps("d8-system").Create(context.TODO(), manifest.(*apiv1.ConfigMap), v1.CreateOptions{})
				return err
			},
			UpdateFunc: func(manifest interface{}) error {
				_, err := fakeClient.CoreV1().ConfigMaps("d8-system").Update(context.TODO(), manifest.(*apiv1.ConfigMap), v1.UpdateOptions{})
				return err
			},
		}
		if err := task.CreateOrUpdate(); err != nil {
			t.Fatalf("%s: Unexpected error: %v", tc.name, err)
		}

		result := GetClusterInfo(fakeClient)
		if tc.expectedResult != result {
			t.Fatalf("%s: %s\n!=\n%s", tc.name, tc.expectedResult, result)
		}
	}
}
