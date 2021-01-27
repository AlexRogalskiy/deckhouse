---
title: "Модуль priority-class: настройки"
---

По умолчанию — **включен**.

В спецификации пода необходимо установить соответствующий [priorityClassName](https://kubernetes.io/docs/concepts/configuration/pod-priority-preemption/#pod-priority).
Очень важно правильно выставлять `priorityClassName`. Если есть сомнения - спросите коллег.

> Любой установленный `priorityClassName` не уменьшит приоритета пода, т.к. если `priority-class` у пода не установлен, шедулер считает его самым низким — `develop`.

Устанавливаемые модулем priority class'ы (в порядке приоритета от большего к меньшему).
**Внимание!** Нельзя использовать PriorityClasses: `system-cluster-critical`, `node-cluster-critical`, `cluster-medium`, `cluster-low`.

| Priority Class            | Описание                                                                                                                                                            | Значение   |
|---------------------------|---------------------------------------------------------------------------------------------------------------------------------------------------------------------|------------|
| `system-node-critical` | Компоненты кластера, которые обязаны присутствовать на ноде. Также полностью защищает от [эвикта kubelet'ом](https://kubernetes.io/docs/tasks/administer-cluster/out-of-resource/).<br>`node-local-dns`, `node-exporter`, `csi`                             | 2000001000 |
| `system-cluster-critical` | Компоненты кластера, без которых его корректная работа невозможна. Этим PriorityClass в обязательном порядке помечаются MutatingWebhooks и Extension API servers. Также полностью защищает от [эвикта kubelet'ом](https://kubernetes.io/docs/tasks/administer-cluster/out-of-resource/).<br>`kube-dns`, `coredns`, `kube-proxy`, `flannel`, `kube-api-server`, `kube-controller-manager`, `kube-scheduler`, `cluster-autoscaler`, `kube-dns`, `machine-controller-manager`.                             | 2000000000 |
| `production-high`         | Stateful приложения, в production окружении отсутствие которых приводит к полной недоступности сервиса или потере данных (postgresql, memcached, redis, mongo, ...). | 9000       |
| `cluster-medium`          | Компоненты кластера, влияющие на мониторинг (алерты, диагностика) кластера и автоскейлинг. Без мониторинга мы не можем оценить масштабы происшествия, без автоскейлинга мы не сможем дать приложениям необходимые ресурсы.<br>`deckhouse`, `kube-state-metrics`, `madison-proxy`, `node-exporter`, `trickster`, `grafana`, `kube-router`, `monitoring-ping`, `okmeter`, `smoke-mini`                      | 7000       |
| `production-medium`       | Основные stateless приложения в production окружении, которые отвечают за работу сервиса для посетителей.                                                            | 6000       |
| `deployment-machinery`    | Компоненты кластера с помощью которых происходит деплой/билд в кластер (helm, werf).<br>`kube-system/tiller-deploy`                                                                                 | 5000       |
| `production-low`          | Приложения в production окружении (кроны, админки, batch-процессинг, ...), без которых можно прожить некоторое время. Если batch или крон никак нельзя прерывать, то он должен быть в production-medium, а не здесь.                                          | 4000       |
| `staging`                 | Staging окружения для приложений.                                                                                                                                    | 3000       |
| `cluster-low`             | Компоненты кластера, без которых возможна эксплуатация кластера, но которые желательны. <br>`prometheus-operator`, `dashboard`, `dashboard-oauth2-proxy`, `cert-manager`, `prometheus`, `prometheus-longterm`, `kube-scheduler-face-slapper`                                                                              | 2000       |
| `develop` (default)       | Develop окружения для приложений. Класс по умолчанию, если не проставлены иные классы.                                                                               | 1000       |
| `standby`                 | Этот класс не предназначен для приложений. Используется в системных целях для резервирования нод.                                                                      | -1         |
