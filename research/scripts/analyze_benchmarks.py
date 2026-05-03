#!/usr/bin/env python3
"""
Скрипт для анализа результатов бенчмарков и генерации отчётов
"""

import json
import sys
from pathlib import Path
from typing import Dict, List, Any
import statistics

def parse_go_benchmark(output: str) -> List[Dict[str, Any]]:
    """Парсит вывод go test -bench"""
    results = []
    
    for line in output.split('\n'):
        if line.startswith('Benchmark'):
            parts = line.split()
            if len(parts) >= 4:
                name = parts[0].replace('Benchmark', '')
                iterations = int(parts[1])
                ns_per_op = float(parts[2])
                
                result = {
                    'name': name,
                    'iterations': iterations,
                    'ns_per_op': ns_per_op,
                    'ms_per_op': ns_per_op / 1_000_000,
                    'ops_per_sec': 1_000_000_000 / ns_per_op if ns_per_op > 0 else 0
                }
                
                # Дополнительные метрики (если есть)
                if len(parts) > 4:
                    for i in range(4, len(parts), 2):
                        if i + 1 < len(parts):
                            key = parts[i]
                            value = parts[i + 1]
                            result[key] = value
                
                results.append(result)
    
    return results

def analyze_encryption_benchmarks(results: List[Dict[str, Any]]) -> Dict[str, Any]:
    """Анализирует бенчмарки шифрования"""
    encryption_results = [r for r in results if 'Encryption' in r['name']]
    decryption_results = [r for r in results if 'Decryption' in r['name']]
    
    analysis = {
        'encryption': {
            'avg_ms': statistics.mean([r['ms_per_op'] for r in encryption_results]) if encryption_results else 0,
            'min_ms': min([r['ms_per_op'] for r in encryption_results]) if encryption_results else 0,
            'max_ms': max([r['ms_per_op'] for r in encryption_results]) if encryption_results else 0,
            'ops_per_sec': statistics.mean([r['ops_per_sec'] for r in encryption_results]) if encryption_results else 0,
        },
        'decryption': {
            'avg_ms': statistics.mean([r['ms_per_op'] for r in decryption_results]) if decryption_results else 0,
            'min_ms': min([r['ms_per_op'] for r in decryption_results]) if decryption_results else 0,
            'max_ms': max([r['ms_per_op'] for r in decryption_results]) if decryption_results else 0,
            'ops_per_sec': statistics.mean([r['ops_per_sec'] for r in decryption_results]) if decryption_results else 0,
        }
    }
    
    return analysis

def analyze_websocket_benchmarks(results: List[Dict[str, Any]]) -> Dict[str, Any]:
    """Анализирует бенчмарки WebSocket"""
    ws_results = [r for r in results if 'WebSocket' in r['name']]
    
    throughput = [r for r in ws_results if 'Throughput' in r['name']]
    latency = [r for r in ws_results if 'Latency' in r['name']]
    concurrent = [r for r in ws_results if 'Concurrent' in r['name']]
    
    analysis = {
        'throughput': {
            'avg_ms': statistics.mean([r['ms_per_op'] for r in throughput]) if throughput else 0,
            'messages_per_sec': statistics.mean([r['ops_per_sec'] for r in throughput]) if throughput else 0,
        },
        'latency': {
            'avg_ms': statistics.mean([r['ms_per_op'] for r in latency]) if latency else 0,
            'p50_ms': 0,  # Нужно парсить из дополнительных метрик
            'p95_ms': 0,
            'p99_ms': 0,
        },
        'concurrent_connections': {
            'max_tested': len(concurrent),
            'avg_ms': statistics.mean([r['ms_per_op'] for r in concurrent]) if concurrent else 0,
        }
    }
    
    return analysis

def analyze_database_benchmarks(results: List[Dict[str, Any]]) -> Dict[str, Any]:
    """Анализирует бенчмарки базы данных"""
    db_results = [r for r in results if 'Database' in r['name']]
    
    insert = [r for r in db_results if 'Insert' in r['name']]
    select = [r for r in db_results if 'Select' in r['name']]
    update = [r for r in db_results if 'Update' in r['name']]
    transaction = [r for r in db_results if 'Transaction' in r['name']]
    
    analysis = {
        'insert': {
            'avg_ms': statistics.mean([r['ms_per_op'] for r in insert]) if insert else 0,
            'ops_per_sec': statistics.mean([r['ops_per_sec'] for r in insert]) if insert else 0,
        },
        'select': {
            'avg_ms': statistics.mean([r['ms_per_op'] for r in select]) if select else 0,
            'ops_per_sec': statistics.mean([r['ops_per_sec'] for r in select]) if select else 0,
        },
        'update': {
            'avg_ms': statistics.mean([r['ms_per_op'] for r in update]) if update else 0,
            'ops_per_sec': statistics.mean([r['ops_per_sec'] for r in update]) if update else 0,
        },
        'transaction': {
            'avg_ms': statistics.mean([r['ms_per_op'] for r in transaction]) if transaction else 0,
            'ops_per_sec': statistics.mean([r['ops_per_sec'] for r in transaction]) if transaction else 0,
        }
    }
    
    return analysis

def generate_markdown_report(analysis: Dict[str, Any], output_file: Path):
    """Генерирует Markdown отчёт"""
    
    report = f"""# Отчёт о производительности Zenthril

Дата: {Path(__file__).stat().st_mtime}

## 📊 Шифрование (AES-256-GCM)

### Encryption
- Среднее время: {analysis['encryption']['encryption']['avg_ms']:.3f} ms
- Минимальное: {analysis['encryption']['encryption']['min_ms']:.3f} ms
- Максимальное: {analysis['encryption']['encryption']['max_ms']:.3f} ms
- Операций в секунду: {analysis['encryption']['encryption']['ops_per_sec']:.0f}

### Decryption
- Среднее время: {analysis['encryption']['decryption']['avg_ms']:.3f} ms
- Минимальное: {analysis['encryption']['decryption']['min_ms']:.3f} ms
- Максимальное: {analysis['encryption']['decryption']['max_ms']:.3f} ms
- Операций в секунду: {analysis['encryption']['decryption']['ops_per_sec']:.0f}

## 🔌 WebSocket

### Пропускная способность
- Среднее время на сообщение: {analysis['websocket']['throughput']['avg_ms']:.3f} ms
- Сообщений в секунду: {analysis['websocket']['throughput']['messages_per_sec']:.0f}

### Задержка
- Среднее: {analysis['websocket']['latency']['avg_ms']:.3f} ms
- P50: {analysis['websocket']['latency']['p50_ms']:.3f} ms
- P95: {analysis['websocket']['latency']['p95_ms']:.3f} ms
- P99: {analysis['websocket']['latency']['p99_ms']:.3f} ms

### Одновременные подключения
- Максимум протестировано: {analysis['websocket']['concurrent_connections']['max_tested']}
- Среднее время: {analysis['websocket']['concurrent_connections']['avg_ms']:.3f} ms

## 💾 База данных (PostgreSQL)

### INSERT
- Среднее время: {analysis['database']['insert']['avg_ms']:.3f} ms
- Операций в секунду: {analysis['database']['insert']['ops_per_sec']:.0f}

### SELECT
- Среднее время: {analysis['database']['select']['avg_ms']:.3f} ms
- Операций в секунду: {analysis['database']['select']['ops_per_sec']:.0f}

### UPDATE
- Среднее время: {analysis['database']['update']['avg_ms']:.3f} ms
- Операций в секунду: {analysis['database']['update']['ops_per_sec']:.0f}

### Транзакции
- Среднее время: {analysis['database']['transaction']['avg_ms']:.3f} ms
- Операций в секунду: {analysis['database']['transaction']['ops_per_sec']:.0f}

## 📈 Выводы

### Сильные стороны:
- Быстрое шифрование/дешифрование (AES-256-GCM)
- Низкая задержка WebSocket соединений
- Эффективная работа с базой данных

### Области для оптимизации:
- [Добавить после анализа]

### Сравнение с аналогами:
- Signal: [данные]
- Matrix: [данные]
- Telegram: [данные]

## 🎯 Рекомендации

1. **Масштабирование**: [рекомендации]
2. **Оптимизация**: [рекомендации]
3. **Мониторинг**: [рекомендации]
"""
    
    output_file.write_text(report, encoding='utf-8')
    print(f"✅ Отчёт сохранён: {output_file}")

def main():
    if len(sys.argv) < 2:
        print("Usage: python analyze_benchmarks.py <benchmark_output.txt>")
        sys.exit(1)
    
    input_file = Path(sys.argv[1])
    if not input_file.exists():
        print(f"❌ Файл не найден: {input_file}")
        sys.exit(1)
    
    # Читаем результаты бенчмарков
    output = input_file.read_text(encoding='utf-8')
    results = parse_go_benchmark(output)
    
    if not results:
        print("❌ Не удалось распарсить результаты бенчмарков")
        sys.exit(1)
    
    print(f"✅ Найдено {len(results)} бенчмарков")
    
    # Анализируем результаты
    analysis = {
        'encryption': analyze_encryption_benchmarks(results),
        'websocket': analyze_websocket_benchmarks(results),
        'database': analyze_database_benchmarks(results),
    }
    
    # Сохраняем JSON
    json_output = input_file.with_suffix('.json')
    json_output.write_text(json.dumps(analysis, indent=2), encoding='utf-8')
    print(f"✅ JSON сохранён: {json_output}")
    
    # Генерируем Markdown отчёт
    md_output = input_file.with_suffix('.md')
    generate_markdown_report(analysis, md_output)

if __name__ == '__main__':
    main()
