module github.com/deckhouse/deckhouse/candictl

go 1.15

require (
	github.com/BurntSushi/toml v0.3.1
	github.com/Masterminds/sprig/v3 v3.1.0
	github.com/alecthomas/template v0.0.0-20190718012654-fb15b899a751 // indirect
	github.com/alecthomas/units v0.0.0-20190924025748-f65c72e2690d // indirect
	github.com/fatih/color v1.9.0
	github.com/flant/logboek v0.3.4
	github.com/flant/shell-operator v1.0.0-beta.13
	github.com/fsnotify/fsnotify v1.4.7
	github.com/go-openapi/spec v0.19.3
	github.com/go-openapi/strfmt v0.19.3
	github.com/go-openapi/validate v0.19.7
	github.com/google/uuid v1.1.1
	github.com/hashicorp/go-multierror v1.0.0
	github.com/iancoleman/strcase v0.0.0-20191112232945-16388991a334
	github.com/peterbourgon/mergemap v0.0.0-20130613134717-e21c03b7a721
	github.com/pkg/errors v0.9.0
	github.com/prometheus/client_golang v1.0.0
	github.com/prometheus/client_model v0.0.0-20190129233127-fd36f4220a90
	github.com/sirupsen/logrus v1.4.2
	github.com/stretchr/testify v1.5.1
	golang.org/x/crypto v0.0.0-20200414173820-0848c9571904
	gopkg.in/alecthomas/kingpin.v2 v2.2.6
	gopkg.in/satori/go.uuid.v1 v1.2.0
	gopkg.in/yaml.v2 v2.2.8
	gopkg.in/yaml.v3 v3.0.0-20200615113413-eeeca48fe776
	k8s.io/api v0.17.0
	k8s.io/apiextensions-apiserver v0.17.0
	k8s.io/apimachinery v0.17.0
	k8s.io/client-go v0.17.0
	k8s.io/klog v1.0.0
	sigs.k8s.io/yaml v1.1.1-0.20191128155103-745ef44e09d6
)

// not working, need to migrate to github.com/alecthomas/kingpin in shell-operator and others
//replace gopkg.in/alecthomas/kingpin.v2 => github.com/flant/kingpin v1.3.8-0.20200415155012-da8c62ac9989

// replace with master branch to work with single dash
replace gopkg.in/alecthomas/kingpin.v2 => github.com/alecthomas/kingpin v1.3.8-0.20200323085623-b6657d9477a6

//replace github.com/flant/shell-operator => ../../shell-operator
