@echo off
REM Скрипт для запуска всех бенчмарков на Windows

echo.
echo ========================================
echo 🚀 Запуск всех бенчмарков Zenthril...
echo ========================================
echo.

REM Создаём папки для результатов
if not exist "research\benchmarks" mkdir research\benchmarks
if not exist "research\data\raw" mkdir research\data\raw
if not exist "research\results" mkdir research\results

REM Проверка переменных окружения
if "%DB_URL%"=="" (
    echo ⚠️  DB_URL не установлен. Бенчмарки БД будут пропущены.
    echo    Установите: set DB_URL=postgres://user:pass@localhost/zenthril
    echo.
    set SKIP_DB=true
) else (
    echo ✅ DB_URL установлен
    set SKIP_DB=false
)

cd backend

REM 1. Бенчмарки шифрования
echo.
echo 📊 Запуск бенчмарков шифрования...
go test -bench=Encryption -benchmem -benchtime=3s ./benchmarks/ > ..\research\benchmarks\encryption_%date:~-4,4%%date:~-10,2%%date:~-7,2%_%time:~0,2%%time:~3,2%%time:~6,2%.txt
echo ✅ Шифрование завершено
echo.

REM 2. Бенчмарки WebSocket
echo 📊 Запуск бенчмарков WebSocket...
go test -bench=WebSocket -benchmem -benchtime=3s ./benchmarks/ > ..\research\benchmarks\websocket_%date:~-4,4%%date:~-10,2%%date:~-7,2%_%time:~0,2%%time:~3,2%%time:~6,2%.txt
echo ✅ WebSocket завершено
echo.

REM 3. Бенчмарки базы данных (если DB_URL установлен)
if "%SKIP_DB%"=="false" (
    echo 📊 Запуск бенчмарков базы данных...
    go test -bench=Database -benchmem -benchtime=3s ./benchmarks/ > ..\research\benchmarks\database_%date:~-4,4%%date:~-10,2%%date:~-7,2%_%time:~0,2%%time:~3,2%%time:~6,2%.txt
    echo ✅ База данных завершена
    echo.
)

REM 4. Все бенчмарки вместе
echo 📊 Запуск всех бенчмарков...
go test -bench=. -benchmem -benchtime=3s ./benchmarks/... > ..\research\benchmarks\all_%date:~-4,4%%date:~-10,2%%date:~-7,2%_%time:~0,2%%time:~3,2%%time:~6,2%.txt
echo ✅ Все бенчмарки завершены
echo.

cd ..

REM 5. Анализ результатов (если Python установлен)
where python >nul 2>nul
if %ERRORLEVEL% EQU 0 (
    echo 📈 Анализ результатов...
    
    REM Находим последний файл с результатами
    for /f "delims=" %%i in ('dir /b /o-d research\benchmarks\all_*.txt 2^>nul') do (
        set LATEST_RESULTS=research\benchmarks\%%i
        goto :found
    )
    :found
    
    if defined LATEST_RESULTS (
        python research\scripts\analyze_benchmarks.py "%LATEST_RESULTS%"
        echo ✅ Анализ завершён
    ) else (
        echo ⚠️  Файлы результатов не найдены
    )
) else (
    echo ⚠️  Python не установлен. Пропускаем анализ.
)

echo.
echo ========================================
echo 🎉 Все бенчмарки завершены!
echo ========================================
echo.
echo Результаты сохранены в:
echo   📁 research\benchmarks\
echo.
echo Следующие шаги:
echo   1. Просмотрите результаты в research\benchmarks\
echo   2. Запустите нагрузочные тесты: cd research\load_test ^&^& k6 run k6_load_test.js
echo   3. Проанализируйте метрики: http://localhost:8080/metrics
echo.

pause
