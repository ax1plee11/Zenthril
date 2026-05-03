# Contributing to Zenthril / Участие в разработке Zenthril

**[English](#english)** | **[Русский](#русский)**

---

## English

Thank you for your interest in contributing to Zenthril! 🎉

### 🐛 Reporting Bugs

Found a bug? Here's how to report it:

1. **Check existing issues** first: https://github.com/ax1plee11/Zenthril/issues
2. **Create a new issue** using the "Bug Report" template
3. **Include**:
   - Clear description of the problem
   - Steps to reproduce
   - Expected vs actual behavior
   - Screenshots (if applicable)
   - Your environment (OS, browser, version)
   - Console logs (F12 → Console)

**Where to report:**
- 🐛 **Bugs**: [GitHub Issues](https://github.com/ax1plee11/Zenthril/issues/new?template=bug_report.md)
- 💡 **Feature requests**: [GitHub Issues](https://github.com/ax1plee11/Zenthril/issues/new?template=feature_request.md)
- 🔒 **Security issues**: Email **ax1plee@gmail.com** (do NOT create public issue)

### 💡 Suggesting Features

Have an idea? We'd love to hear it!

1. Check if it's already suggested
2. Create a new issue using "Feature Request" template
3. Describe the problem it solves
4. Explain your proposed solution

### 🔒 Security Vulnerabilities

**IMPORTANT:** Do NOT report security vulnerabilities publicly!

**Contact:**
- 📧 Email: **ax1plee@gmail.com**
- 🔐 GitHub Security Advisories: https://github.com/ax1plee11/Zenthril/security/advisories

We will:
- Respond within 48 hours
- Fix critical issues within 7 days
- Credit you in the security advisory (if desired)

### 🛠️ Contributing Code

Want to contribute code? Great!

1. **Fork** the repository
2. **Create a branch**: `git checkout -b feature/your-feature-name`
3. **Make your changes**
4. **Test** your changes:
   ```bash
   # Backend
   cd backend
   go test ./...
   
   # Client
   cd client
   npm test
   npm run lint
   ```
5. **Commit** with clear message: `feat: add amazing feature`
6. **Push** to your fork
7. **Create a Pull Request**

### 📝 Code Style

- **Go**: Follow standard Go conventions, use `gofmt`
- **TypeScript/React**: Follow the existing code style
- **Commits**: Use [Conventional Commits](https://www.conventionalcommits.org/)
  - `feat:` - new feature
  - `fix:` - bug fix
  - `docs:` - documentation
  - `style:` - formatting
  - `refactor:` - code refactoring
  - `test:` - tests
  - `chore:` - maintenance

### 🌍 Translations

Want to add a new language?

1. Copy `client/src/i18n/locales/en.json`
2. Translate all strings
3. Add language to `client/src/i18n/index.ts`
4. Add flag emoji to `languages` array
5. Submit a Pull Request

### 📞 Contact

- 📧 Email: **ax1plee@gmail.com**
- 💬 GitHub Discussions: https://github.com/ax1plee11/Zenthril/discussions
- 🐛 Issues: https://github.com/ax1plee11/Zenthril/issues

---

## Русский

Спасибо за интерес к участию в разработке Zenthril! 🎉

### 🐛 Сообщение об ошибках

Нашли баг? Вот как сообщить:

1. **Проверьте существующие issues**: https://github.com/ax1plee11/Zenthril/issues
2. **Создайте новый issue** используя шаблон "Bug Report"
3. **Включите**:
   - Чёткое описание проблемы
   - Шаги для воспроизведения
   - Ожидаемое и фактическое поведение
   - Скриншоты (если есть)
   - Ваше окружение (ОС, браузер, версия)
   - Логи консоли (F12 → Console)

**Куда сообщать:**
- 🐛 **Баги**: [GitHub Issues](https://github.com/ax1plee11/Zenthril/issues/new?template=bug_report.md)
- 💡 **Идеи**: [GitHub Issues](https://github.com/ax1plee11/Zenthril/issues/new?template=feature_request.md)
- 🔒 **Уязвимости**: Email **ax1plee@gmail.com** (НЕ создавайте публичный issue)

### 💡 Предложение функций

Есть идея? Мы будем рады услышать!

1. Проверьте, не предложена ли она уже
2. Создайте новый issue используя шаблон "Feature Request"
3. Опишите проблему, которую это решит
4. Объясните ваше предлагаемое решение

### 🔒 Уязвимости безопасности

**ВАЖНО:** НЕ сообщайте об уязвимостях публично!

**Контакты:**
- 📧 Email: **ax1plee@gmail.com**
- 🔐 GitHub Security Advisories: https://github.com/ax1plee11/Zenthril/security/advisories

Мы:
- Ответим в течение 48 часов
- Исправим критические проблемы в течение 7 дней
- Укажем вас в благодарностях (если хотите)

### 🛠️ Участие в коде

Хотите внести код? Отлично!

1. **Форкните** репозиторий
2. **Создайте ветку**: `git checkout -b feature/название-функции`
3. **Внесите изменения**
4. **Протестируйте** изменения:
   ```bash
   # Backend
   cd backend
   go test ./...
   
   # Client
   cd client
   npm test
   npm run lint
   ```
5. **Закоммитьте** с понятным сообщением: `feat: добавить крутую функцию`
6. **Запушьте** в свой форк
7. **Создайте Pull Request**

### 📝 Стиль кода

- **Go**: Следуйте стандартным Go конвенциям, используйте `gofmt`
- **TypeScript/React**: Следуйте существующему стилю кода
- **Коммиты**: Используйте [Conventional Commits](https://www.conventionalcommits.org/)
  - `feat:` - новая функция
  - `fix:` - исправление бага
  - `docs:` - документация
  - `style:` - форматирование
  - `refactor:` - рефакторинг кода
  - `test:` - тесты
  - `chore:` - обслуживание

### 🌍 Переводы

Хотите добавить новый язык?

1. Скопируйте `client/src/i18n/locales/en.json`
2. Переведите все строки
3. Добавьте язык в `client/src/i18n/index.ts`
4. Добавьте флаг эмодзи в массив `languages`
5. Отправьте Pull Request

### 📞 Контакты

- 📧 Email: **ax1plee@gmail.com**
- 💬 GitHub Discussions: https://github.com/ax1plee11/Zenthril/discussions
- 🐛 Issues: https://github.com/ax1plee11/Zenthril/issues

---

## Thank you! / Спасибо!

Your contributions make Zenthril better for everyone! 🚀

Ваш вклад делает Zenthril лучше для всех! 🚀
