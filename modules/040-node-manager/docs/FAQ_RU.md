---
title: "Управление узлами: FAQ"
search: добавить ноду в кластер, добавить узел в кластер, настроить узел с GPU, эфемерные узлы
---

## Как автоматически добавить статичный узел в кластер?

Чтобы добавить новый узел в статичный кластер необходимо:
- Создать `NodeGroup` с необходимыми параметрами (`nodeType` может быть `Static` или `CloudStatic`) или использовать уже существующую. К примеру создадим `NodeGroup` с именем `example`.
- Получить скрипт для установки и настройки узла: `kubectl -n d8-cloud-instance-manager get secret manual-bootstrap-for-example -o json | jq '.data."bootstrap.sh"' -r`
- Перед настройкой kubernetes на узле убедитесь, что вы выполнили все необходимые действия для корректной работы узла в кластере:
  - Добавили в `/etc/fstab` все необходимые маунты (nfs, ceph, ...)
  - Установили на узел `ceph-common` нужной версии или еще какие-то пакеты
  - Настроили сеть для коммуникации узлов в кластере
- Зайти на новый узел по ssh и выполнить команду из секрета: `echo <base64> | base64 -d | bash`

## Как завести узел под управление node-manager?

Чтобы завести узел под управление `node-manager`:
- Создать `NodeGroup` с необходимыми параметрами (`nodeType` может быть `Static` или `CloudStatic`) или использовать уже существующую. К примеру создадим `NodeGroup` с именем `nodes`.
- Получить скрипт для установки и настройки узла: `kubectl -n d8-cloud-instance-manager  get secret manual-bootstrap-for-nodes-o json | jq '.data."adopt.sh"' -r`
- Зайти на новый узел по ssh и выполнить команду из секрета: `echo <base64> | base64 -d | bash`

## Как изменить node-group у статичного узла?

Чтобы перенести существующий статичный узел из одной node-group в другую, необходимо изменить у узла лейбл группы:
```shell
kubectl label node --overwrite <node_name> node.deckhouse.io/group=<group_name>
```

Изменения не будут применены мгновенно. Обновлением состояния объектов NodeGroup занимается один из хуков deckhouse, который подписывается на изменения узлов.

## Как вывести узел из-под управления node-manager?

- Остановить сервис и таймер bashible: `systemctl stop bashible.timer bashible.service`
- Удалить скрипты bashible: `rm -rf /var/lib/bashible`
- Удалить с узла аннотации и лейблы:
```shell
kubectl annotate node <node_name> node.deckhouse.io/configuration-checksum- update.node.deckhouse.io/waiting-for-approval- update.node.deckhouse.io/disruption-approved- update.node.deckhouse.io/disruption-required- update.node.deckhouse.io/approved- update.node.deckhouse.io/draining- update.node.deckhouse.io/drained-
kubectl label node <node_name> node.deckhouse.io/group-
```

## Как зачистить узел для последующего ввода в кластер?

1. Остановим все сервисы.
    ```shell
    systemctl stop kubernetes-api-proxy.service kubernetes-api-proxy-configurator.service kubernetes-api-proxy-configurator.timer
    systemctl stop bashible.service bashible.timer
    systemctl stop kubelet.service
    systemctl stop docker
    ```
2. Удалим маунты.
   ```shell
   for i in $(mount -t tmpfs | grep /var/lib/kubelet | cut -d " " -f3); do umount $i ; done
   ```
3. Удалим директории и файлы.
   ```shell
   rm -rf /var/lib/bashible 
   rm -rf /etc/kubernetes
   rm -rf /var/lib/kubelet 
   rm -rf /var/lib/docker 
   rm -rf /etc/cni
   rm -rf /var/lib/cni
   rm -rf /var/lib/etcd
   rm -rf /etc/systemd/system/kubernetes-api-proxy*
   rm -rf /etc/systemd/system/bashible*
   rm -rf /etc/systemd/system/sysctl-tuner*
   rm -rf /etc/systemd/system/kubelet*
   ```
4. Удалим интерфейсы.
   ```shell
   ifconfig cni0 down
   ifconfig flannel.1 down
   ifconfig docker0 down
   ip link delete cni0
   ip link delete flannel.1
   ```
5. Очистка systemd:
   ```shell
   systemctl daemon-reload
   systemctl reset-failed
   ```
6. Запустим обратно Docker.
   ```shell
   systemctl start docker
   ```
7. [Запустим](#как-автоматически-добавить-статичный-узел-в-кластер) `bootstrap.sh`.
8. Включим все сервисы обратно.
   ```shell
   systemctl start kubelet.service
   systemctl start kubernetes-api-proxy.service kubernetes-api-proxy-configurator.service kubernetes-api-proxy-configurator.timer
   systemctl start bashible.service bashible.timer
   ```

## Как понять, что что-то пошло не так?

Модуль `node-manager` создает на каждом узле сервис `bashible`, и его логи можно посмотреть при помощи: `journalctl -fu bashible`.

## Как посмотреть, что в данный момент выполняется на узле при ее создании?

Если мы хотим узнать, что происходит на узле (к примеру она долго создается), то можно посмотреть логи `cloud-init` для этого необходимо:
- Найти узел, которая сейчас бутстрапится: `kubectl -n d8-cloud-instance-manager  get machine | grep Pending`
- Посмотреть информацию о `machine`: `kubectl -n d8-cloud-instance-manager describe machine kube-2-worker-01f438cf-757f758c4b-r2nx2`
В дескрайбе мы увидим такую информацию:
```shell
Status:
  Bootstrap Status:
    Description:   Use 'nc 192.168.199.115 8000' to get bootstrap logs.
    Tcp Endpoint:  192.168.199.115
```
- Выполнить команду `nc 192.168.199.115 8000`, тогда вы увидите логи `cloud-init` и увидите на чем зависла настройка узла.

Логи первоначальной настройки узла находятся в `/var/log/cloud-init-output.log`.

## Как настроить узел с GPU?

Если у вас есть узел с GPU и вы хотите настроить docker для работы с `node-manager`, то вам необходимо выполнить все настройки на узле по [документации](https://github.com/NVIDIA/k8s-device-plugin#quick-start).

Создать `NodeGroup` с такими параметрами:
```shell
  docker:
    manage: false
  operatingSystem:
    manageKernel: false
```

После чего, добавить узел под управление `node-manager`.

## Какие параметры NodeGroup к чему приводят?

| Параметр NG                   | Disruption update    | Перезаказ узлов | Рестарт kubelet |
|-------------------------------|----------------------|---------------|-----------------|
| operatingSystem.manageKernel  | + (true) / - (false) | -             | -               |
| kubelet.maxPods               | -                    | -             | +               |
| kubelet.rootDir               | -                    | -             | +               |
| docker.maxConcurrentDownloads | +                    | -             | +               |
| docker.manage                 | + (true) / - (false) | -             | -               |
| nodeTemplate                  | -                    | -             | -               |
| chaos                         | -                    | -             | -               |
| kubernetesVersion             | -                    | -             | +               |
| static                        | -                    | -             | +               |
| disruptions                   | -                    | -             | -               |
| cloudInstances.classReference | -                    | +             | -               |

Подробно о всех параметрах можно прочитать в описании custom resource [NodeGroup]({{ "/modules/040-node-manager/cr.html#nodegroup" | true_relative_url }})

В случае изменения параметра `instancePrefix` в конфигурации deckhouse не будет происходить `RollingUpdate`. Deckhouse создаст новые `MachineDeployment`, а старые удалит.

## Как перекатить эфемерные машины в облаке с новой конфигурацией?

При изменении конфигурации Deckhouse (как в модуле node-manager, так и в любом из облачных провайдеров) виртуальные машины не будут перезаказаны. Перекат происходит только после изменения `InstanceClass` или `NodeGroup` объектов.

Для того, чтобы форсированно перекатить все Machines, следует добавить/изменить аннотацию `manual-rollout-id` в `NodeGroup`: `kubectl annotate NodeGroup имя_ng "manual-rollout-id=$(uuidgen)" --overwrite`.

## Как выделить узлы под специфические нагрузки?

> ⛔ Запрещено использование домена `deckhouse.io` в ключах `labels` и `taints` у `NodeGroup`. Он зарезервирован для компонентов **Deckhouse**. Отдайте предпочтение в пользу ключей `dedicated` или `dedicated.client.com`.

Для решений данной задачи существуют два механизма:
- Установка меток в `NodeGroup` `spec.nodeTemplate.labels`, для последующего использования их в `Pod` [spec.nodeSelector](https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/) или [spec.affinity.nodeAffinity](https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/#node-affinity). Указывает, какие именно узлы будут выбраны планировщиком для запуска целевого приложения.
- Установка ограничений в `NodeGroup` `spec.nodeTemplate.taints`, с дальнейшим снятием их в `Pod` [spec.tolerations](https://kubernetes.io/docs/concepts/scheduling-eviction/taint-and-toleration/). Запрещает исполнение не разрешенных явно приложений на этих узлах.

> ℹ Deckhouse по умолчанию толерейтит ключ `dedicated`, поэтому рекомендуется использовать ключ `dedicated` с любым `value` для тейнтов на ваших выделенных узлах.️
> Если необходимо использовать произвольные ключи для `taints` (например, `dedicated.client.com`), то нужно добавить в `ConfigMap` `d8-system/deckhouse` в секцию `global.modules.placement.customTolerationKeys` значение ключа. Таким образом мы разрешим системным компонентам (например `cni-flannel`) выезжать на эти выделенные узлы.

Подробности [в статье на Habr](https://habr.com/ru/company/flant/blog/432748/).

## Как выделить узлы под системные компоненты?

### Фронтенд
Для **Ingress**-контроллеров используйте `NodeGroup` со следующей конфигурацией:

```yaml
  nodeTemplate:
    labels:
      node-role.deckhouse.io/frontend: ""
    taints:
    - effect: NoExecute
      key: dedicated.deckhouse.io
      value: frontend
```

### Системные
`NodeGroup` для компонентов подсистем Deckhouse, будут с такими параметрами:

```yaml
  nodeTemplate:
    labels:
      node-role.deckhouse.io/system: ""
    taints:
    - effect: NoExecute
      key: dedicated.deckhouse.io
      value: system
```

## Как ускорить заказ узлов в облаке при горизонтальном масштабировании приложений?

Самое действенное — держать в кластере некоторое количество "подогретых" узлов, которые позволят новым репликам ваших приложений запускаться мгновенно. Очевидным минусом данного решения будут дополнительные расходы на содержание этих узлов.

Необходимые настройки целевой `NodeGroup` будут следующие:
1. Указать абсолютное количество "подогретых" узлов (или процент от максимального количества узлов в этой группе) в параметре `cloudInstances.standby`.
1. При наличии, дополнительных служебных компонентов (не обслуживаемых Deckhouse, например DaemonSet `filebeat`) для этих узлов — задать их суммарное потребление ресурсов в параметре `standbyHolder.notHeldResources`.
1. Для работы этой функции требуется, чтобы как минимум один узел из группы уже был запущен в кластере. Иными словами должна быть либо одна реплика приложения, либо количество узлов для этой группы `cloudInstances.minPerZone` должно быть `1`.

Пример:
```yaml
  cloudInstances:
    maxPerZone: 10
    minPerZone: 1
    standby: 10%
    standbyHolder:
      notHeldResources:
        cpu: 300m
        memory: 2Gi
```
## Как выключить machine-controller-manager в случае выполнения потенциально деструктивных изменений в кластере?
>⛔ ***Внимание!!!*** Использовать эту настройку только тогда, когда вы четко понимаете зачем это необходимо !!!

Установить параметр:
```yaml
mcmEmergencyBrake: true
```
## Как восстановить master-узел, если kubelet не может загрузить компоненты control-plane?

Подобная ситуация может возникнуть, если в кластере с одним master-узлом на нем были удалены образы
компонентов `control-plane` (например, удалена директория `/var/lib/docker` при использовании docker или `/var/lib/containerd` при использовании containerd). В этом случае при рестарте `kubelet` он не сможет выкачать образы `control-plane`-компонентов, поскольку на master-узле нет параметров авторизации в `registry.deckhouse.io`.

Как восстановить:
### Docker
Для восстановления работоспособности мастера нужно в любом рабочем кластере под управлением Deckhouse 
выполнить команду:

```
kubectl -n d8-system get secrets deckhouse-registry -o json |
jq -r '.data.".dockerconfigjson"' | base64 -d |
jq -r 'del(.auths."registry.deckhouse.io".username, .auths."registry.deckhouse.io".password)'
```
Вывод команды нужно скопировать и добавить его в файл `/root/.docker/config.json` на поврежденном мастере.
Далее, на поврежденном мастере нужно загрузить образы `control-plane` компонентов:
```
for image in $(grep "image:" /etc/kubernetes/manifests/* | awk '{print $3}'); do
  docker pull $image
done
```
После загрузки образов необходимо перезапустить `kubelet`.
После восстановления работоспособности мастера нужно убрать внесенные в файл `/root/.docker/config.json`  изменения !!!

### Containerd
Для восстановления работоспособности мастера нужно в любом рабочем кластере под управлением Deckhouse
выполнить команду:

```
kubectl -n d8-system get secrets deckhouse-registry -o json |
jq -r '.data.".dockerconfigjson"' | base64 -d |
jq -r '.auths."registry.deckhouse.io".auth'
```
Вывод команды нужно скопировать и присвоить переменной AUTH на поврежденном мастере.
Далее, на поврежденном мастере нужно загрузить образы `control-plane` компонентов:
```
for image in $(grep "image:" /etc/kubernetes/manifests/* | awk '{print $3}'); do
  crictl pull --auth $AUTH $image
done
```
После загрузки образов необходимо перезапустить `kubelet`.
