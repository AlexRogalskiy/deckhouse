---
title: "Модуль ingress-nginx: настройки"
---

Модуль по умолчанию **включен** в кластерах начиная с версии 1.14. Для выключения добавьте в CM `d8-system/deckhouse`:
```yaml
ingressNginxEnabled: "false"
```

## Параметры

* `defaultControllerVersion` — версия контроллера ingress-nginx, которая будет использоваться для всех контроллеров по умолчанию, если небыл задан параметр `controllerVersion` в IngressNginxController CR.
    * По умолчанию `0.33`,
    * Доступные варианты: `0.25`, `0.26`, `0.33`, `0.46`.


Конфигурация Ingress-контроллеров выполняется с помощью Custom Resource [IngressNginxController](cr.html#ingressnginxcontroller).
