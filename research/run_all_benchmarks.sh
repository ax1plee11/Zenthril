#!/bin/bash
# Скрипт для запуска всех бенчмарков и сбора результатов

set -e

echo "🚀 Запуск всех бенчмарков Zenthril..."
echo "======================================"
echo ""

# Создаём папки для результатов
mkdir -p research/benchmarks
mkdir -p research/data/raw
mkdir -p research/results

# Цвета для вывода
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Проверка переменных окружения
if [ -z "$DB_URL" ]; then
    echo -e "${YELLOW}⚠️  DB_URL не установлен. Бенчмарки БД будут пропущены.${NC}"
    echo "   Установите: export DB_URL='postgres://user:pass@localhost/zenthril'"
    echo ""
    SKIP_DB=true
else
    echo -e "${GREEN}✅ DB_URL установлен${NC}"
    SKIP_DB=false
fi

cd backend

# 1. Бенчмарки шифрования
echo -e "${BLUE}📊 Запуск бенчмарков шифрования...${NC}"
go test -bench=Encryption -benchmem -benchtime=3s ./benchmarks/ > ../research/benchmarks/encryption_$(date +%Y%m%d_%H%M%S).txt
echo -e "${GREEN}✅ Шифрование завершено${NC}"
echo ""

# 2. Бенчмарки WebSocket
echo -e "${BLUE}📊 Запуск бенчмарков WebSocket...${NC}"
go test -bench=WebSocket -benchmem -benchtime=3s ./benchmarks/ > ../research/benchmarks/websocket_$(date +%Y%m%d_%H%M%S).txt
echo -e "${GREEN}✅ WebSocket завершено${NC}"
echo ""

# 3. Бенчмарки базы данных (если DB_URL установлен)
if [ "$SKIP_DB" = false ]; then
    echo -e "${BLUE}📊 Запуск бенчмарков базы данных...${NC}"
    go test -bench=Database -benchmem -benchtime=3s ./benchmarks/ > ../research/benchmarks/database_$(date +%Y%m%d_%H%M%S).txt
    echo -e "${GREEN}✅ База данных завершена${NC}"
    echo ""
fi

# 4. Все бенчмарки вместе
echo -e "${BLUE}📊 Запуск всех бенчмарков...${NC}"
if [ "$SKIP_DB" = false ]; then
    go test -bench=. -benchmem -benchtime=3s ./benchmarks/... > ../research/benchmarks/all_$(date +%Y%m%d_%H%M%S).txt
else
    go test -bench=. -benchmem -benchtime=3s -run=^$ ./benchmarks/ | grep -v Database > ../research/benchmarks/all_$(date +%Y%m%d_%H%M%S).txt || true
fi
echo -e "${GREEN}✅ Все бенчмарки завершены${NC}"
echo ""

cd ..

# 5. Анализ результатов (если Python установлен)
if command -v python3 &> /dev/null; then
    echo -e "${BLUE}📈 Анализ результатов...${NC}"
    
    # Находим последний файл с результатами
    LATEST_RESULTS=$(ls -t research/benchmarks/all_*.txt 2>/dev/null | head -1)
    
    if [ -n "$LATEST_RESULTS" ]; then
        python3 research/scripts/analyze_benchmarks.py "$LATEST_RESULTS"
        echo -e "${GREEN}✅ Анализ завершён${NC}"
    else
        echo -e "${YELLOW}⚠️  Файлы результатов не найдены${NC}"
    fi
else
    echo -e "${YELLOW}⚠️  Python3 не установлен. Пропускаем анализ.${NC}"
fi

echo ""
echo "======================================"
echo -e "${GREEN}🎉 Все бенчмарки завершены!${NC}"
echo ""
echo "Результаты сохранены в:"
echo "  📁 research/benchmarks/"
echo ""
echo "Следующие шаги:"
echo "  1. Просмотрите результаты в research/benchmarks/"
echo "  2. Запустите нагрузочные тесты: cd research/load_test && k6 run k6_load_test.js"
echo "  3. Проанализируйте метрики: http://localhost:8080/metrics"
echo ""
