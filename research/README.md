# Performance Research & Benchmarking

Tools and scripts for performance analysis and academic research.

## 🎯 Research Goals

1. **Encryption Performance** - measure AES-256-GCM overhead
2. **WebSocket Scalability** - test concurrent connection limits
3. **Database Performance** - optimize PostgreSQL queries
4. **Load Testing** - stability under high load
5. **Comparative Analysis** - benchmark against similar platforms

### 1. Запуск бенчмарков

```bash
cd backend

# Все бенчмарки
go test -bench=. -benchmem ./benchmarks/... > ../research/benchmarks/results.txt

# Только шифрование
go test -bench=Encryption -benchmem ./benchmarks/ > ../research/benchmarks/encryption.txt

# Только WebSocket
go test -bench=WebSocket -benchmem ./benchmarks/ > ../research/benchmarks/websocket.txt

# Только база данных (требует DB_URL)
DB_URL="postgres://user:pass@localhost/zenthril" go test -bench=Database -benchmem ./benchmarks/ > ../research/benchmarks/database.txt
```

### 2. Нагрузочное тестирование

```bash
cd research/load_test

# Установка k6 (если ещё не установлен)
# Windows: choco install k6
# macOS: brew install k6
# Linux: см. https://k6.io/docs/getting-started/installation/

# Запуск теста
k6 run k6_load_test.js

# С кастомными параметрами
k6 run --vus 100 --duration 5m k6_load_test.js

# Сохранение результатов
k6 run --out json=results.json k6_load_test.js
```

### 3. Анализ результатов

```bash
cd research/scripts

# Анализ бенчмарков
python analyze_benchmarks.py ../benchmarks/results.txt

# Это создаст:
# - results.json (JSON данные)
# - results.md (Markdown отчёт)
```

### 4. Мониторинг в реальном времени

Запустите сервер и откройте:

```bash
# JSON метрики
http://localhost:8080/metrics

# Prometheus формат
http://localhost:8080/metrics/prometheus
```

## 📊 Метрики

### Собираемые метрики:

#### WebSocket
- `active_connections` - активные подключения
- `total_connections` - всего подключений
- `total_messages` - всего сообщений
- `message_latency_p50/p95/p99` - задержка доставки

#### Шифрование
- `encryption_ops` - операций шифрования
- `decryption_ops` - операций дешифрования
- `avg_encryption_ms` - среднее время шифрования
- `avg_decryption_ms` - среднее время дешифрования

#### База данных
- `db_queries` - количество запросов
- `avg_db_query_ms` - среднее время запроса
- `db_errors` - ошибки БД
- `db_latency_p50/p95/p99` - задержка запросов

#### HTTP
- `http_requests` - количество запросов
- `http_errors` - ошибки (4xx, 5xx)
- `avg_http_response_ms` - среднее время ответа
- `http_latency_p50/p95/p99` - задержка ответов

## 🎯 Цели исследования

### Для магистерской работы нужно:

1. **Производительность шифрования**
   - Сравнить AES-256-GCM с другими алгоритмами
   - Измерить накладные расходы E2EE
   - Оптимизировать узкие места

2. **Масштабируемость WebSocket**
   - Максимальное количество одновременных подключений
   - Задержка при разной нагрузке
   - Потребление ресурсов

3. **Производительность БД**
   - Оптимизация запросов
   - Влияние индексов
   - Параллельные запросы

4. **Сравнение с аналогами**
   - Signal Protocol
   - Matrix Synapse
   - Telegram MTProto

## 📈 Ожидаемые результаты

### Минимальные показатели:
- ✅ 100+ одновременных пользователей
- ✅ P95 < 500ms
- ✅ Шифрование < 1ms на сообщение
- ✅ Ошибок < 1%

### Целевые показатели:
- 🎯 500+ одновременных пользователей
- 🎯 P95 < 200ms
- 🎯 Шифрование < 0.5ms
- 🎯 Ошибок < 0.1%

### Отличные показатели:
- 🏆 1000+ одновременных пользователей
- 🏆 P95 < 100ms
- 🏆 Шифрование < 0.1ms
- 🏆 Ошибок < 0.01%

## 📝 Создание отчёта

### Структура отчёта для диссертации:

1. **Методология**
   - Описание тестового окружения
   - Инструменты и метрики
   - Сценарии тестирования

2. **Результаты**
   - Графики производительности
   - Таблицы с метриками
   - Сравнение с аналогами

3. **Анализ**
   - Узкие места
   - Оптимизации
   - Масштабируемость

4. **Выводы**
   - Достигнутые показатели
   - Научная новизна
   - Практическая значимость

## 🔧 Инструменты

### Установленные:
- Go benchmarks (встроенные)
- k6 (нагрузочное тестирование)
- Python (анализ данных)

### Рекомендуемые:
- Grafana + Prometheus (визуализация)
- pgAdmin (мониторинг PostgreSQL)
- Redis Commander (мониторинг Redis)

## 📚 Дополнительные материалы

- [Нагрузочное тестирование](load_test/README.md)
- [Бенчмарки Go](https://pkg.go.dev/testing#hdr-Benchmarks)
- [k6 документация](https://k6.io/docs/)
- [Prometheus метрики](https://prometheus.io/docs/concepts/metric_types/)

## 🤝 Вклад в исследование

Если вы хотите добавить новые тесты или метрики:

1. Добавьте бенчмарк в `backend/benchmarks/`
2. Обновите скрипты анализа в `scripts/`
3. Задокументируйте результаты в `results/`
4. Создайте Pull Request

## 📞 Контакты

Вопросы по исследованию: ax1plee@gmail.com
