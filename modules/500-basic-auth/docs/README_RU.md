---
title: "Модуль basic-auth"
---

Модуль устанавливает сервис для базовой авторизации.

**Внимание!** Модуль не предназначен для высоких нагрузок.

Конфигурация
------------

### Включение модуля

Модуль по умолчанию **выключен**. Для включения добавьте в CM `deckhouse`:

```yaml
data:
  basicAuthEnabled: "true"
```

### Что нужно настраивать?
Обязательных настроек нет.
По умолчанию создается location `/` с пользователем `admin`.

### Параметры
* `highAvailability` — ручное включение/отключение режима отказоустойчивости. По умолчанию режим отказоустойчивости определяется автоматически. Смотри [подробнее]({{ "/overview.html#параметры" | true_relative_url }}) про режим отказоустойчивости для модулей.
* `locations` — если нам необходимо создать несколько location'ов для разных приложений с разной авторизацией, то добавляем данный параметр.
    * `location` — это location, для которого будут определяться `whitelist` и `users`, в конфиге nginx `root` заменяется на `/`.
    * `whitelist` — список IP адресов и подсетей для которых разрешена авторизация без логина/пароля.
    * `users` — список пользователей в формате `username: "password"`.
* `nodeSelector` — как в Kubernetes в `spec.nodeSelector` у pod'ов.
    * Если ничего не указано — будет [использоваться автоматика](/overview.html#выделение-узлов-под-определенный-вид-нагрузки).
    * Можно указать `false`, чтобы не добавлять никакой nodeSelector.
* `tolerations` — как в Kubernetes в `spec.tolerations` у pod'ов.
    * Если ничего не указано — будет [использоваться автоматика](/overview.html#выделение-узлов-под-определенный-вид-нагрузки).
    * Можно указать `false`, чтобы не добавлять никакие toleration'ы.

### Пример конфигурации:

```yaml
basicAuthEnabled: "true"
basicAuth: |
  locations:
  - location: "/"
    whitelist:
      - 1.1.1.1
    users:
      username: "password"
  nodeSelector:
    node-role/example: ""
  tolerations:
  - key: dedicated
    operator: Equal
    value: example
```

### Использование
Просто добавляем подобную аннотацию к ингрессу:

`nginx.ingress.kubernetes.io/auth-url: "http://basic-auth.kube-basic-auth.svc.cluster.local/"`
