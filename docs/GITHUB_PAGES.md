# GitHub Pages Setup

## Автоматическая публикация веб-версии

Проект настроен для автоматической публикации на GitHub Pages при каждом push в `main`.

## Настройка (один раз)

1. Открой репозиторий на GitHub: https://github.com/ax1plee11/Zenthril

2. Перейди в **Settings** → **Pages**

3. В разделе **Build and deployment**:
   - **Source**: выбери `GitHub Actions`

4. Сохрани настройки

## После настройки

После следующего push в `main` ветку:
- GitHub Actions автоматически соберёт проект
- Опубликует на GitHub Pages
- Приложение будет доступно по адресу:

```
https://ax1plee11.github.io/Zenthril/
```

## Поделиться с друзьями

Просто отправь ссылку:
```
https://ax1plee11.github.io/Zenthril/
```

Друзья смогут:
- ✅ Открыть приложение в браузере
- ✅ Зарегистрироваться и войти
- ✅ Использовать все функции (если backend запущен)
- ✅ Переключать язык интерфейса

## Важно: Backend

GitHub Pages публикует только **frontend** (клиентскую часть).

Для полноценной работы нужен **backend**:

### Вариант 1: Локальный backend
```bash
# Запусти backend локально
docker compose up -d
cd backend
go run main.go
```

Затем обнови `client/.env`:
```env
VITE_API_BASE=http://localhost:8080
```

### Вариант 2: Публичный backend

Разверни backend на хостинге (см. [DEPLOYMENT.md](DEPLOYMENT.md)):
- Heroku
- Railway
- DigitalOcean
- AWS/GCP/Azure

Затем обнови переменную окружения в GitHub Actions:
```yaml
env:
  VITE_API_BASE: https://your-backend.com
```

## Обновление

При каждом push в `main`:
1. GitHub Actions автоматически пересобирает проект
2. Обновляет GitHub Pages
3. Изменения появляются через 1-2 минуты

## Проверка статуса

Проверь статус деплоя:
- Перейди в **Actions** на GitHub
- Найди workflow "Deploy to GitHub Pages"
- Посмотри логи сборки

## Альтернативы GitHub Pages

Если нужен более мощный хостинг:

### Vercel (рекомендуется)
```bash
npm i -g vercel
cd client
vercel
```

Преимущества:
- Автоматический HTTPS
- Быстрый CDN
- Автоматические превью для PR
- Бесплатный план

### Netlify
```bash
npm i -g netlify-cli
cd client
netlify deploy
```

### Cloudflare Pages
- Подключи репозиторий
- Укажи build command: `cd client && npm run build`
- Укажи publish directory: `client/dist`

## Кастомный домен

Если хочешь использовать свой домен (например, `zenthril.com`):

1. Купи домен (Namecheap, GoDaddy, etc.)
2. В настройках GitHub Pages укажи домен
3. Добавь DNS записи:
   ```
   CNAME: www -> ax1plee11.github.io
   A: @ -> 185.199.108.153
   A: @ -> 185.199.109.153
   A: @ -> 185.199.110.153
   A: @ -> 185.199.111.153
   ```

## Troubleshooting

### "Page not found" после деплоя
- Проверь что в Settings → Pages выбран `GitHub Actions`
- Подожди 2-3 минуты после деплоя

### Белый экран
- Проверь консоль браузера (F12)
- Убедись что `base` в vite.config.ts правильный
- Проверь что все файлы загрузились (Network tab)

### Backend не подключается
- Проверь CORS настройки на backend
- Убедись что `VITE_API_BASE` указывает на правильный URL
- Проверь что backend доступен извне (не localhost)
