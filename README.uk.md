# Zenthril

**[English](README.md)** | **[Русский](README.ru.md)** | **[Українська](README.uk.md)**

---

Децентралізований месенджер з федеративною архітектурою та наскрізним шифруванням.

## Технологічний стек

- **Backend**: Go + PostgreSQL + Redis
- **Client**: Tauri + Vite + React/TypeScript
- **Шифрування**: X25519 + AES-256-GCM (E2EE)
- **Аутентифікація**: JWT + Argon2id
- **UI**: Tailwind CSS + shadcn/ui + Glass Minimal Design
- **i18n**: Підтримка кількох мов (EN, RU, UK)

## Можливості

✨ **Сучасний дизайн**
- Glass Minimal UI в стилі Apple/Notion
- Темна тема з ефектами glassmorphism
- Плавні анімації та переходи
- Адаптивна верстка

🌍 **Інтернаціоналізація**
- Автоматичне визначення мови
- Підтримка англійської, російської, української
- Легко додати нові мови

🔒 **Безпека та приватність**
- Наскрізне шифрування (E2EE)
- Федеративна архітектура
- Без відстеження та збору даних
- Відкритий вихідний код

💬 **Комунікація**
- Текстові канали та особисті повідомлення
- Голосові канали (WebRTC)
- Підтримка GIF (Tenor/Giphy)
- Обмін повідомленнями в реальному часі (WebSocket)

## Документація

- [SECURITY.md](SECURITY.md) — Як повідомляти про вразливості
- [docs/PRIVACY.md](docs/PRIVACY.md) — Чернетка політики конфіденційності
- [docs/DEPLOYMENT.md](docs/DEPLOYMENT.md) — Публічний хостинг: TLS, `VITE_API_BASE`, CORS/WS, бекапи
- [BUILD_INSTRUCTIONS.md](BUILD_INSTRUCTIONS.md) — Як зібрати desktop-додаток (.exe)

## Якість коду (локально)

**Backend** (`backend/`):

```bash
go vet ./...
go test ./... -count=1
# go test -race ./...   # на Linux/macOS і Windows amd64; на win/386 недоступний
```

Лінтер: [golangci-lint](https://golangci-lint.run/) з конфігом `backend/.golangci.yml` (той самий запускається в CI).

**Client** (`client/`):

```bash
npm run lint
npm run test
npm run test:coverage
npm run build
```

## Структура проекту

```
zenthril/
├── backend/          # Go-сервер (вузол федеративної мережі)
│   ├── config/       # Конфігурація через змінні оточення
│   ├── migrations/   # SQL-міграції PostgreSQL
│   └── main.go       # Точка входу HTTP-сервера
├── client/           # Tauri + Vite + React/TypeScript desktop-клієнт
│   ├── src/
│   │   ├── components/  # React-компоненти
│   │   ├── i18n/        # Інтернаціоналізація (EN, RU, UK)
│   │   └── store/       # Управління станом
│   └── src-tauri/    # Tauri (Rust) desktop-обгортка
├── docs/             # Документація
├── docker-compose.yml
└── .env.example
```

## Швидкий старт

```bash
# 1. Скопіювати змінні оточення
cp .env.example .env

# 2. Запустити backend-сервіси (PostgreSQL + Redis)
docker compose up -d

# 3. Встановити залежності клієнта
cd client
npm install

# 4. Скопіювати змінні оточення клієнта
cp .env.example .env

# 5. Запустити dev-сервер
npm run dev -- --host 0.0.0.0 --port 1420
```

Відкрий `http://localhost:1420/`. Backend буде доступний на `http://localhost:8080/`.

## Змінні оточення

### Backend (корінь репозиторію)

- `DB_URL` (обов'язково) — Рядок підключення до PostgreSQL
- `REDIS_URL` (за замовчуванням: `redis://localhost:6379`)
- `JWT_SECRET` (обов'язково) — Секретний ключ для JWT-токенів
- `HTTP_ADDR` (за замовчуванням: `:8080`)
- `CORS_ALLOWED_ORIGINS` (опціонально) — Дозволені CORS origins
- `WS_ALLOWED_ORIGINS` (опціонально) — Дозволені WebSocket origins
- `ADMIN_USER_IDS` (опціонально, UUID через кому) — Доступ до `/api/v1/admin/*` (включаючи global ban)

### Client (папка `client/`)

Скопіюй `client/.env.example` → `client/.env`.

- `VITE_API_BASE` (для production-збірки) — Origin backend'у, наприклад `https://api.example.com` (див. [docs/DEPLOYMENT.md](docs/DEPLOYMENT.md))
- `VITE_TENOR_KEY` (опціонально) — API-ключ Tenor для пошуку GIF
- `VITE_GIPHY_KEY` (опціонально) — API-ключ Giphy для пошуку GIF

## Збірка desktop-додатку (Windows)

Для збірки desktop-додатку на Windows потрібні **Visual Studio Build Tools** (з `link.exe`):

1. Завантаж [Visual Studio Build Tools 2022](https://visualstudio.microsoft.com/downloads/#build-tools-for-visual-studio-2022)
2. Вибери "Desktop development with C++"
3. Встанови (~6 ГБ)
4. Перезапусти термінал

Потім зібери:

```bash
cd client
npm run tauri build
```

Файл `.exe` буде в `client/src-tauri/target/release/bundle/`.

Детальні інструкції в [BUILD_INSTRUCTIONS.md](BUILD_INSTRUCTIONS.md).

## Скріншоти

### Glass Minimal дизайн
![Екран авторизації](https://via.placeholder.com/800x500?text=Auth+Screen+with+Language+Switcher)

### Підтримка кількох мов
- 🇬🇧 English
- 🇷🇺 Русский
- 🇺🇦 Українська

Мова автоматично визначається з налаштувань браузера і може бути змінена вручну.

## Участь у розробці

Ми вітаємо внесок у проект! Будь ласка, прочитайте [SECURITY.md](SECURITY.md) перед повідомленням про вразливості.

## Ліцензія

Цей проект з відкритим вихідним кодом. Див. файл LICENSE для деталей.

## Подяки

- Дизайн натхненний Apple та Notion
- UI-компоненти від [shadcn/ui](https://ui.shadcn.com/)
- Іконки від [Lucide](https://lucide.dev/)
