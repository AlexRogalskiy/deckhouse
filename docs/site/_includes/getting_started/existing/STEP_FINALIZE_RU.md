<script type="text/javascript" src='{{ assets["getting-started.js"].digest_path }}'></script>
<script type="text/javascript" src='{{ assets["getting-started-access.js"].digest_path }}'></script>

На данном этапе вы установили Deckhousе, и он уже взял на себя часть функций по управлению кластером. Например, процесс добавления узла в кластер стал проще — подробнее об этом можно узнать <a href="/ru/documentation/v1/modules/040-node-manager/faq.html#как-автоматически-добавить-статичный-узел-в-кластер">в документации</a>.

<blockquote>
<p>Некоторые компоненты Deckhouse по умолчанию не работают на master-узле. Если вам необходимо разрешить компонентам Deckhouse работать на master-узле, снимите с master-узла taint, выполнив следующую команду:</p>
{% snippetcut %}
```bash
kubectl patch nodegroup master --type json -p '[{"op": "remove", "path": "/spec/nodeTemplate/taints"}]'
```
{% endsnippetcut %}
</blockquote>

Остается только <strong>создать DNS-записи</strong> для доступа в веб-интерфейсы кластера:
<ul>
  <li>Выясните публичный IP-адрес узла, на котором работает Ingress-контроллер.</li>
  <li>Если у вас есть возможность добавить DNS-запись используя DNS-сервер:
    <ul>
      <li>Если ваш шаблон DNS-имен кластера является <a href="https://en.wikipedia.org/wiki/Wildcard_DNS_record">wildcard
        DNS-шаблоном</a> (например - <code>%s.kube.my</code>), то добавьте соответствующую wildcard A-запись со значением публичного IP-адреса, который вы получили выше.
      </li>
      <li>
        Если ваш шаблон DNS-имен кластера <strong>НЕ</strong> является <a
              href="https://en.wikipedia.org/wiki/Wildcard_DNS_record">wildcard DNS-шаблоном</a> (например - <code>%s-kube.company.my</code>),
        то добавьте А или CNAME-записи со значением публичного IP-адреса, который вы
        получили выше, для следующих DNS-имен сервисов Deckhouse в вашем кластере:
        <div class="highlight">
<pre class="highlight">
<code example-hosts>dashboard.example.com
deckhouse.example.com
dex.example.com
grafana.example.com
kubeconfig.example.com
status.example.com
upmeter.example.com</code>
</pre>
        </div>
      </li>
    </ul>
  </li>

  <li><p>Если вы <strong>не</strong> имеете под управлением DNS-сервер: добавьте статические записи соответствия имен конкретных сервисов публичному IP-адресу узла, на котором работает Ingress-контроллер.</p><p>Например, на персональном Linux-компьютере, с которого необходим доступ к сервисам Deckhouse, выполните следующую команду (укажите ваш публичный IP-адрес в переменной <code>PUBLIC_IP</code>) для добавления записей в файл <code>/etc/hosts</code> (для Windows используйте файл <code>%SystemRoot%\system32\drivers\etc\hosts</code>):</p>
{% snippetcut selector="export-ip" %}
```shell
export PUBLIC_IP="<PUBLIC_IP>"
```
{% endsnippetcut %}

<p>Добавьте необходимые записи в файл <code>/etc/hosts</code>>:</p>

{% snippetcut selector="example-hosts" %}
```shell
sudo -E bash -c "cat <<EOF >> /etc/hosts
$PUBLIC_IP dashboard.example.com
$PUBLIC_IP deckhouse.example.com
$PUBLIC_IP dex.example.com
$PUBLIC_IP grafana.example.com
$PUBLIC_IP kubeconfig.example.com
$PUBLIC_IP status.example.com
$PUBLIC_IP upmeter.example.com
EOF
"
```
{% endsnippetcut %}
</li></ul>

<script type="text/javascript">
$(document).ready(function () {
    generate_password();
    update_parameter('dhctl-user-password-hash', 'password', '<GENERATED_PASSWORD_HASH>', null, null);
    update_parameter('dhctl-user-password-hash', null, '<GENERATED_PASSWORD_HASH>', null, '[user-yml]');
    update_parameter('dhctl-user-password', null, '<GENERATED_PASSWORD>', null, '[user-yml]');
    update_parameter('dhctl-user-password', null, '<GENERATED_PASSWORD>', null, 'code span.c1');
    update_domain_parameters();
});

</script>
