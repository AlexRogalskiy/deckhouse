---
title: "Модуль user-authz: настройки"
---

Модуль по умолчанию **включен**. Для выключения добавьте в ConfigMap `deckhouse`:

```yaml
data:
  userAuthzEnabled: "false"
```

> **Внимание!** Мы категорически не рекомендуем создавать Pod'ы и ReplicaSet'ы – эти объекты являются второстепенными и должны создаваться из других контроллеров. Доступ к созданию и изменению Pod'ов и ReplicaSet'ов полностью отсутствует.

> **Внимание!** Режим multi-tenancy (авторизация по namespace) в данный момент реализован по временной схеме и **не гарантирует безопасность**! Если webhook, который реализовывает систему авторизации по какой-то причине упадёт, авторизация по namespace (опции `allowAccessToSystemNamespaces` и `limitNamespaces` в CR) перестанет работать и пользователи получат доступы во все namespace. После восстановления доступности webhook'а все вернется на свои места.

## Параметры

* `enableMultiTenancy` — включить авторизацию по namespace.
  * Так как данная опция реализована через [плагин авторизации Webhook](https://kubernetes.io/docs/reference/access-authn-authz/webhook/), то потребуется дополнительная [настройка kube-apiserver](usage.html#настройка-kube-apiserver). Для автоматизации этого процесса используйте модуль [control-plane-manager](../../modules/040-control-plane-manager/).
  * Значение по умолчанию – `false` (то есть multi-tenancy отключен).
  * **Доступно только в версии Enterprise Edition.**
* `controlPlaneConfigurator` — настройки параметров для модуля автоматической настройки kube-apiserver [control-plane-manager](../../modules/040-control-plane-manager/).
  * `enabled` — передавать ли в control-plane-manager параметры для настройки authz-webhook (см. [параметры control-plane-manager'а](../../modules/040-control-plane-manager/configuration.html#параметры)).
    * При выключении этого параметра, модуль control-plane-manager будет считать, что по умолчанию Webhook-авторизация выключена и, соответственно, если не будет дополнительных настроек, то control-plane-manager будет стремиться вычеркнуть упоминания Webhook-плагина из манифеста. Даже если вы настроите манифест вручную.
    * По умолчанию `true`.

Вся настройка прав доступа происходит с помощью [Custom Resources](cr.html).
