# Как поделиться Zenthril с друзьями

## 🌐 Вариант 1: Веб-версия (самый простой)

После настройки GitHub Pages (см. [docs/GITHUB_PAGES.md](docs/GITHUB_PAGES.md)):

**Просто отправь ссылку:**
```
https://ax1plee11.github.io/Zenthril/
```

Друзья смогут открыть приложение прямо в браузере!

---

## 📦 Вариант 2: Desktop-приложение (.exe)

### Для Windows:

1. **Собери .exe файл** (см. [BUILD_INSTRUCTIONS.md](BUILD_INSTRUCTIONS.md))
2. **Найди файл:**
   ```
   client/src-tauri/target/release/bundle/nsis/Zenthril_0.1.0_x64-setup.exe
   ```
3. **Загрузи на файлообменник:**
   - Google Drive
   - Dropbox
   - WeTransfer
   - Mega.nz
4. **Отправь ссылку другу**

Друг просто скачает и установит приложение!

---

## 💻 Вариант 3: Исходный код (для разработчиков)

**Отправь ссылку на GitHub:**
```
https://github.com/ax1plee11/Zenthril
```

Друг сможет:
```bash
# 1. Клонировать репозиторий
git clone https://github.com/ax1plee11/Zenthril.git
cd Zenthril

# 2. Запустить backend
cp .env.example .env
docker compose up -d

# 3. Запустить клиент
cd client
npm install
cp .env.example .env
npm run dev
```

Откроет `http://localhost:1420/`

---

## 🚀 Вариант 4: Публичный сервер

### Быстрый деплой на Railway (бесплатно, 500 часов/месяц)

#### Шаг 1: Создай проект на Railway

1. Открой https://railway.app/
2. Войди через GitHub
3. Нажми **"New Project"**
4. Выбери **"Deploy from GitHub repo"**
5. Выбери репозиторий **Zenthril**

#### Шаг 2: Добавь базы данных

**PostgreSQL:**
1. Нажми **"+ New"** → **"Database"** → **"PostgreSQL"**
2. Railway автоматически создаст переменную `DATABASE_URL`

**Redis:**
1. Нажми **"+ New"** → **"Database"** → **"Redis"**
2. Railway автоматически создаст переменную `REDIS_URL`

#### Шаг 3: Настрой переменные окружения

В настройках backend-сервиса добавь:

```env
# JWT Secret (сгенерируй случайную строку)
JWT_SECRET=your-super-secret-jwt-key-min-32-characters-long

# CORS для GitHub Pages
CORS_ALLOWED_ORIGINS=https://ax1plee11.github.io

# WebSocket origins
WS_ALLOWED_ORIGINS=https://ax1plee11.github.io
```

#### Шаг 4: Получи URL backend

После деплоя Railway даст тебе URL типа:
```
https://zenthril-production.up.railway.app
```

#### Шаг 5: Обнови GitHub Pages

Добавь переменную окружения в `.github/workflows/deploy-pages.yml`:

```yaml
- name: Build
  working-directory: ./client
  run: npm run build
  env:
    VITE_BASE: /Zenthril/
    VITE_API_BASE: https://zenthril-production.up.railway.app
```

#### Шаг 6: Готово!

Теперь твои друзья могут:
```
https://ax1plee11.github.io/Zenthril/
```

И всё будет работать! 🎉

---

### Альтернативы Railway:

**Render.com** (бесплатно, но засыпает после 15 мин неактивности):
- https://render.com/
- Аналогично Railway, но медленнее

**Fly.io** (бесплатно, 3 VM):
- https://fly.io/
- Быстрее, но сложнее настройка

**DigitalOcean** ($5/месяц):
- https://www.digitalocean.com/
- Самый надёжный вариант

---

Если хочешь, чтобы друзья могли использовать твой сервер:

### 1. Разверни backend на хостинге

**Бесплатные варианты:**
- [Railway](https://railway.app/) - 500 часов/месяц бесплатно
- [Render](https://render.com/) - бесплатный план
- [Fly.io](https://fly.io/) - бесплатный план

**Платные варианты:**
- DigitalOcean ($5/месяц)
- AWS/GCP/Azure
- Heroku

### 2. Настрой домен (опционально)

Купи домен на:
- Namecheap (~$10/год)
- GoDaddy
- Cloudflare

### 3. Настрой HTTPS

Используй:
- Let's Encrypt (бесплатно)
- Cloudflare (бесплатно)

### 4. Поделись ссылкой

```
https://zenthril.yourdomain.com
```

---

## 📱 Вариант 5: QR-код

Создай QR-код со ссылкой:

1. Открой https://www.qr-code-generator.com/
2. Вставь ссылку на твой Zenthril
3. Скачай QR-код
4. Отправь друзьям

Друзья просто отсканируют QR-код телефоном!

---

## 🎯 Рекомендации

### Для обычных пользователей:
✅ **Веб-версия** (Вариант 1) - самый простой способ

### Для друзей-разработчиков:
✅ **GitHub** (Вариант 3) - могут посмотреть код и запустить локально

### Для постоянного использования:
✅ **Desktop-приложение** (Вариант 2) - работает без браузера

### Для большой аудитории:
✅ **Публичный сервер** (Вариант 4) - все подключаются к одному серверу

---

## 🔒 Безопасность

При публичном деплое:

1. **Измени JWT_SECRET** в `.env`
2. **Настрой CORS** правильно
3. **Используй HTTPS** (обязательно!)
4. **Настрой rate limiting**
5. **Включи мониторинг**

См. [docs/DEPLOYMENT.md](docs/DEPLOYMENT.md) для деталей.

---

## 💡 Советы

### Для веб-версии:
- Убедись что backend доступен извне (не localhost)
- Настрой CORS для твоего домена
- Используй HTTPS для backend

### Для desktop-приложения:
- Подпиши .exe файл (чтобы Windows не ругался)
- Создай установщик с красивым UI
- Добавь автообновление

### Для публичного сервера:
- Настрой бэкапы базы данных
- Мониторь использование ресурсов
- Настрой логирование
- Добавь rate limiting

---

## 📊 Сравнение вариантов

| Вариант | Сложность | Скорость | Стоимость | Для кого |
|---------|-----------|----------|-----------|----------|
| Веб-версия | ⭐ | ⚡⚡⚡ | Бесплатно | Все |
| Desktop .exe | ⭐⭐ | ⚡⚡ | Бесплатно | Windows |
| GitHub | ⭐⭐⭐ | ⚡ | Бесплатно | Разработчики |
| Публичный сервер | ⭐⭐⭐⭐ | ⚡⚡⚡ | $0-50/мес | Большая аудитория |

---

## 🆘 Помощь

Если друзья столкнулись с проблемами:

1. Проверь [docs/GITHUB_PAGES.md](docs/GITHUB_PAGES.md)
2. Проверь [BUILD_INSTRUCTIONS.md](BUILD_INSTRUCTIONS.md)
3. Проверь [docs/DEPLOYMENT.md](docs/DEPLOYMENT.md)
4. Создай Issue на GitHub

---

## 🎉 Готово!

Теперь ты знаешь все способы поделиться Zenthril с друзьями!

Выбери подходящий вариант и вперёд! 🚀
