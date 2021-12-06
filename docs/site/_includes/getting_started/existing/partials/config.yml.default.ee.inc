# [<en>] section for bootstrapping the Deckhouse cluster (InitConfiguration)
# [<en>] version of the Deckhouse API
# [<ru>] секция первичной инициализации кластера Deckhouse (InitConfiguration)
# [<ru>] используемая версия API Deckhouse
apiVersion: deckhouse.io/v1
# [<en>] type of the configuration section
# [<ru>] тип секции конфигурации
kind: InitConfiguration
# [<en>] Deckhouse parameters
# [<ru>] секция с параметрами Deckhouse
deckhouse:
  # [<en>] address of the Docker registry where the Deckhouse images are located
  # [<ru>] адрес Docker registry с образами Deckhouse
  imagesRepo: registry.deckhouse.io/deckhouse/ee
  # [<en>] a special string with your token to access Docker registry (generated automatically for your license token)
  # [<ru>] строка с ключом для доступа к Docker registry (сгенерировано автоматически для вашего токена доступа)
  registryDockerCfg: <YOUR_ACCESS_STRING_IS_HERE>
  # [<en>] the release channel in use
  # [<ru>] используемый канал обновлений
  releaseChannel: Stable
  # [<en>] the Managed bundle is used when installing Deckhouse in an existing cluster
  # [<ru>] при установке Deckhouse в существующий кластер используется вариант поставки — Managed
  bundle: Managed
  configOverrides:
    global:
      modules:
        # [<en>] template that will be used for system apps domains within the cluster
        # [<en>] e.g., Grafana for %s.example.com will be available as grafana.example.com
        # [<ru>] шаблон, который будет использоваться для составления адресов системных приложений в кластере
        # [<ru>] например, Grafana для %s.example.com будет доступна на домене grafana.example.com
        publicDomainTemplate: "%s.example.com"
        # [<en>] if there are no master nodes in the cluster (cloud managed control-plane) or they have special taints
        # [<en>] specify them to allow Deckhouse components to start or delete the `placement` block
        # [<ru>] если в кластере отсутствуют master-узлы (cloud managed control-plane) или у них особые taint
        # [<ru>] укажите их, чтобы позволить запуститься компонентам Deckhouse или удалите блок `placement`
        # [<en>] you might consider changing this
        # [<ru>] возможно, захотите изменить
        placement:
          customTolerationKeys:
          - CriticalAddonsOnly
    cniFlannelEnabled: false
    userAuthnCrdEnabled: true
    userAuthnEnabled: true
    userAuthn:
      publishAPI:
        enable: false
      controlPlaneConfigurator:
        enabled: false
    prometheusMadisonIntegrationEnabled: false
    controlPlaneConfiguratorEnabled: false
    userAuthzEnabled: true
    userAuthz:
      controlPlaneConfigurator:
        enabled: false
    # [<en>] Enable if you don't have an Ingress controller installed in your cluster yet
    # [<ru>] Включите, если у вас в кластере еще не установлен Ingress-контроллер
    # [<en>] you might consider changing this
    # [<ru>] возможно, захотите изменить
    nginxIngressEnabled: false
