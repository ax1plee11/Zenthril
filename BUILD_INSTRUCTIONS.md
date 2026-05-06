# Инструкция по сборке Zenthril.exe

## Вариант 1: Сборка через GitHub Actions (БЕЗ установки Build Tools)

Самый простой способ - использовать GitHub Actions для автоматической сборки.

### Шаги:

1. **Открой свой репозиторий на GitHub:**
   ```
   https://github.com/ax1plee11/Zenthril
   ```

2. **Перейди во вкладку "Actions"**

3. **Найди workflow "Build Desktop App"**

4. **Нажми "Run workflow"** → **"Run workflow"**

5. **Подожди 10-15 минут** пока GitHub соберёт приложение

6. **Скачай готовые файлы:**
   - Перейди в завершённый workflow
   - В разделе "Artifacts" скачай:
     - `zenthril-windows-installer` - установщик .exe
     - `zenthril-windows-portable` - portable .exe (без установки)

### Преимущества:
- ✅ Не нужно устанавливать Build Tools (~6 ГБ)
- ✅ Собирается на серверах GitHub (бесплатно)
- ✅ Автоматически собирает для Windows, Linux, macOS
- ✅ Можно скачать готовый .exe

---

## Вариант 2: Локальная сборка (требует Build Tools)

Если хочешь собирать локально на своём компьютере.

## Требования

### 1. Visual Studio Build Tools
Для сборки Tauri приложения на Windows нужен компилятор C++.

**Установка:**
1. Скачай [Visual Studio Build Tools 2022](https://visualstudio.microsoft.com/downloads/#build-tools-for-visual-studio-2022)
2. Запусти установщик
3. Выбери **"Desktop development with C++"**
4. Нажми "Install" (потребуется ~6 ГБ)
5. Перезапусти компьютер после установки

### 2. Проверка установки
```bash
# Проверь что Rust установлен
rustc --version

# Проверь что Node.js установлен
node --version
npm --version
```

## Сборка приложения

### Шаг 1: Установи зависимости
```bash
cd client
npm install
```

### Шаг 2: Собери .exe файл
```bash
npm run tauri build
```

Процесс займёт 5-10 минут при первой сборке.

### Шаг 3: Найди готовый .exe
После успешной сборки файлы будут здесь:
```
client/src-tauri/target/release/bundle/
├── msi/           # Установщик .msi
│   └── Zenthril_0.1.0_x64_en-US.msi
└── nsis/          # Установщик .exe
    └── Zenthril_0.1.0_x64-setup.exe
```

Также будет standalone .exe:
```
client/src-tauri/target/release/zenthril.exe
```

## Альтернатива: Dev-режим

Если не хочешь собирать, можешь запустить в dev-режиме:

```bash
cd client
npm run tauri dev
```

Это откроет desktop-приложение без сборки .exe

## Веб-версия

Или просто запусти веб-версию:

```bash
cd client
npm run dev
```

Открой http://localhost:1420/ в браузере.

## Возможные ошибки

### "linker `link.exe` not found"
**Решение:** Установи Visual Studio Build Tools (см. выше)

### "failed to run custom build command"
**Решение:** Перезапусти терминал после установки Build Tools

### "error: could not compile"
**Решение:** Обнови Rust:
```bash
rustup update
```

## Что включено в сборку

✅ Glass Minimal дизайн-система
✅ Интернационализация (EN, RU, UK)
✅ shadcn/ui компоненты
✅ Tailwind CSS
✅ End-to-end шифрование
✅ Автоматическое определение языка

## Размер приложения

- Установщик: ~15-20 МБ
- Установленное приложение: ~30-40 МБ
- Первая сборка: ~10 минут
- Последующие сборки: ~2-3 минуты

## Дополнительно

### Изменить иконку приложения
Замени файлы в `client/src-tauri/icons/`

### Изменить название
Отредактируй `client/src-tauri/tauri.conf.json`:
```json
{
  "productName": "Zenthril",
  "version": "0.1.0"
}
```

### Создать portable версию
Добавь в `tauri.conf.json`:
```json
{
  "bundle": {
    "targets": ["portable"]
  }
}
```
