# Rule Authoring — безопасный контракт правила (раздел 07)

Проверочное правило — декларативная конфигурация над зарегистрированным handler-ом сканера.
Правило НЕ содержит произвольного Python, shell-команд или непроверенных сетевых payload-цепочек.

Поля RuleRevision:
- rule_id / version / key (rule_id:version)
- weakness_ids: существующие CWE
- handler_key: идентификатор внутреннего handler-а
- mode / safety_class: passive / active, low-impact (по allowlist)
- preconditions: права владельца, технология, согласие на активные тесты
- request_budget: max_requests, timeout_ms, rate_limit (строгий серверный потолок)
- signals: схема признаков, уровень подтверждения, причины неопределённости
- remediation_ids: рекомендации и ссылки на первоисточники
- status / owner / checksum: draft → reviewed → approved → retired; автор и SHA-256

Пример (образец схемы, не готовое правило):
```yaml
rule_id: VRX-SQLI-001
version: 1.0.0
status: draft
weakness_ids: [CWE-89]
handler_key: passive_error_pattern_v1
mode: passive
safety_class: passive
request_budget:
  max_requests: 0
  timeout_ms: 2500
signals:
  - id: VRX-SIGNAL-DB-ERROR
    verification_level: suspected
remediation_ids: [VRX-FIX-PARAM-QUERIES]
review:
  requires_appsec_approval: true
```

## Статусы доказательств (раздел 07.3)

| Статус | Значение | Как показывать |
|---|---|---|
| suspected | Есть индикаторы; причинность не установлена. | «Возможная проблема», без утверждения об эксплуатации. |
| confirmed | Есть достаточные воспроизводимые признаки в разрешённом тесте. | «Подтверждено», с материалами и временем. |
| inconclusive | Сигналы противоречивы или недоступны. | «Не удалось определить». |
| not_observed | Проверка проведена, индикатор не обнаружен. | Не означает «сайт безопасен». |

Уровень подтверждения сигнала (`verification_level`) в RuleRevision сигналах должен соответствовать одному из этих статусов.

