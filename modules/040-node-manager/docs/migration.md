---
title: "Управление узлами: FAQ"
search: миграция flant.com, dedicated.flant.com, node-role.flant.com, D8DeprecatedNodeSelectorOrTolerationFound, D8DeprecatedNodeGroupLabelOrTaintFound, D8DeprecatedNodeSelectorOrTolerationFoundInCluster, D8DeprecatedNodeGroupLabelOrTaintFoundInCluster
---

##### ✊ Миграция `flant.com` ➡️ `deckhouse.io`

> - `<имя модуля>` – kebab-case имя модуля (`module-name`).
> - `<пустое значение>` – в `tolerations` означает совпадение с любыми значениями, что-то вроде **wildcard**.

###### Соглашения по миграции:
1. В ранее использованных `labels` `node-role.flant.com/(system|frontend|monitoring|<имя модуля>)` домен `flant.com` изменяется на `deckhouse.io`.
1. В ранее использованных `taints`, с ключами `dedicated.flant.com` и значениями `(system|frontend|monitoring|<имя модуля>)` домен `flant.com` изменяется на `deckhouse.io`.
1. В ранее использованных `tolerations`, с ключами `dedicated.flant.com` и значениями `(system|frontend|monitoring|<имя модуля>|<пустое значение>)` домен `flant.com` изменяется на `deckhouse.io`.
1. Остальные `labels` вида `node-role.flant.com/production` или `node-role.flant.com/whatever` могут быть использованы далее без изменений. 
1. Остальные `taints`, с ключами `dedicated.flant.com` и значениями `production` или `whatever` могут быть использованы далее без изменений.
1. Остальные `tolerations`, с ключами `dedicated.flant.com` и значениями `production` или `whatever` могут быть использованы далее без изменений.

###### Последовательность для беспростойной миграции
> ⚠️ Сначала убеждаемся, что приложение выкатывается в новых условиях, потом удаляем старые условия. В противном случае будет простой, что недопустимо для **production**-окружений.

1. Мигрируем ресурсы из алертов `D8DeprecatedNodeSelectorOrTolerationFound`:
   - В ключах `nodeSelector` / `nodeAffinity`, попадающих под выражение `node-role.flant.com/(system|frontend|monitoring|<имя модуля>)` – сменить доменное имя на новое `node-role.deckhouse.io`.
   - Для `tolerations` с ключами `dedicated.flant.com` и значениями `(system|frontend|monitoring|<имя модуля>|<пустое значение>)` **добавить еще один** – с ключом `dedicated.deckhouse.io` и таким же значением.
1. Если есть такой же алерт `D8DeprecatedNodeSelectorOrTolerationFound` для `ConfigMap` `deckhouse`, то требуется внести правки в конфигурации модулей:
   - Переходим в редактирование `cm/deckhouse` `kubectl -n d8-system edit cm deckhouse`.
   - Идём по модулям, описанным в нём.
   - В ключах `nodeSelector`, попадающих под выражение `node-role.flant.com/(system|frontend|monitoring|<имя модуля>)` – сменить доменное имя на новое `node-role.deckhouse.io`.
   - Для `tolerations` с ключами `dedicated.flant.com` и значениями `(system|frontend|monitoring|<имя модуля>|<пустое значение>)` **добавить еще один** – с ключом `dedicated.deckhouse.io` и таким же значением.
   - Сохраняем.
1. ℹ️ Имейте в виду, что на данном этапе алерты не погаснут, так как продолжат срабатывать на `dedicated.flant.com` в `tolerations`. Так и задумано, идём дальше.
1. ⌛️ Дожидаемся, что все приложения успешно запустились с новыми `nodeSelector` / `nodeAffinity` и `tolerations`.
   - Особое внимание следует уделить особо важным компонентам, таким как **Ingress**-контроллеры `watch kubectl -n d8-ingress-nginx get pod`.
   - Не должно остаться подвисших в `Terminating` / `Pending` подов `watch "kubectl get pod -A | grep -v Running"`.
1. 🧐 Все приложения перезапустились успешно? Идём дальше ⤵️.
1. Мигрируем `NodeGroup` из алертов `D8DeprecatedNodeGroupLabelOrTaintFound`:
   - Меняем старый домен в `nodeTemplate.labels` на новый.
   - Меняем старый домен в `nodeTemplate.taints` на новый.
1. 🚮 Удаляем, прежде оставленные, `tolerations` `dedicated.flant.com`:
   - В ресурсах из алертов `D8DeprecatedNodeSelectorOrTolerationFound`.
   - В `ConfigMap` `deckhouse`.
1. ✅ Теперь алерты на ресурсы погаснут.

> ☝️ Если это dev/stage кластер и простой приложений не страшен – для ресурсов этих приложений, в шагах 1 и 2, сразу меняйте `tolerations` на новые. Это позволит избежать второй итерации правок ресурсов из последнего пункта. 
