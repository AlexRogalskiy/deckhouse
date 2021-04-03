module github.com/deckhouse/deckhouse

go 1.13

require (
	github.com/deckhouse/deckhouse/candictl v0.0.0 // use non-existent version for replace
	github.com/benjamintf1/unmarshalledmatchers v0.0.0-20190408201839-bb1c1f34eaea
	github.com/fatih/color v1.9.0
	github.com/flant/addon-operator v1.0.0-rc.1.0.20210318173605-65f01a49f656 // branch: master
	github.com/flant/shell-operator v1.0.1-0.20210401174712-041657a02ccb // branch: master
	github.com/gammazero/deque v0.0.0-20190521012701-46e4ffb7a622
	github.com/ghodss/yaml v1.0.0
	github.com/go-openapi/spec v0.19.3
	github.com/google/go-cmp v0.4.0
	github.com/iancoleman/strcase v0.0.0-20191112232945-16388991a334
	github.com/imdario/mergo v0.3.8
	github.com/kyokomi/emoji v2.1.0+incompatible
	github.com/mohae/deepcopy v0.0.0-20170929034955-c48cc78d4826
	github.com/onsi/ginkgo v1.12.0
	github.com/onsi/gomega v1.9.0
	github.com/otiai10/copy v1.0.2
	github.com/sirupsen/logrus v1.4.2
	github.com/tidwall/gjson v1.3.4
	github.com/tidwall/sjson v1.0.4
	golang.org/x/sys v0.0.0-20200113162924-86b910548bc1
	gopkg.in/evanphx/json-patch.v4 v4.5.0
	gopkg.in/yaml.v2 v2.2.8
	gopkg.in/yaml.v3 v3.0.0-20200615113413-eeeca48fe776
	helm.sh/helm/v3 v3.2.4
	k8s.io/api v0.18.0
	k8s.io/apiextensions-apiserver v0.18.0
	k8s.io/apimachinery v0.18.0
	sigs.k8s.io/yaml v1.2.0
)

replace github.com/deckhouse/deckhouse/candictl => ./candictl

// TODO uncomment when shell-operator migrates to client-go 0.20.0
// TODO remove when https://github.com/helm/helm/pull/8371 will be merged and released.
//replace helm.sh/helm/v3 => github.com/diafour/helm/v3 v3.2.5-0.20200630114452-b734742e3342 // branch: fix_tpl_performance_3_2_4

// TODO remove replaces below when shell-operator migrates to client-go 0.20.0
// TODO remove ./helm-mod directory as well!
replace helm.sh/helm/v3 => ./helm-mod

replace k8s.io/api => k8s.io/api v0.17.0

replace k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.17.0

replace k8s.io/apimachinery => k8s.io/apimachinery v0.17.0

replace k8s.io/client-go => k8s.io/client-go v0.17.0
