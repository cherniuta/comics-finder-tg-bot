# Comics Finder Telegram Bot 🤖📚

**Telegram-бот для поиска комиксов с [xkcd](https://xkcd.com) с микросервисной архитектурой на Go**

![Go](https://img.shields.io/badge/Go-00ADD8?style=for-the-badge&logo=go&logoColor=white)
![Telegram](https://img.shields.io/badge/Telegram-2CA5E0?style=for-the-badge&logo=telegram&logoColor=white)
![Microservices](https://img.shields.io/badge/Microservices-Architecture-6DA55F?style=for-the-badge)

## 🌟 О проекте

- 🔍 Умный поиск по ключевым словам
- 👩‍💼 Разделение ролей (пользователь и администратор)
- 🚀 Высокая производительность за счет индексации
- 🏗 Модульная архитектура для легкого масштабирования, gRPC микросервисы
- 🐳 Сборка через Docker Compose
- 🧪 Покрытие Unit и интеграционными тестами

## 🛠 Технологический стек

```mermaid
graph LR
    A[Telegram Bot] --> B[API Gateway]
    B --> C[Update]
    B --> D[Search]
    C --> G[xkcd API]
    D --> F
    C --> F[(DB)]
    C --> E[Words]
    D --> E[Words]
```
- **Go (Golang)** - основной язык разработки
- **Telegram Bot API** - взаимодействие с пользователями
- **gRPC** - межсервисное взаимодействие
- **PostgreSQL** - база данных
- **Docker + Compose** - средство запуска

## 👾 Команды бота
### Пользователь
```
/start - Приветственное сообщение
/help - Список команд
/search [query] - Поиск комиксов по ключевым словам
/admin - Вход как администратор
```
### Администратор
```
/update - Обновление базы комиксов
/drop - Удалить базу комиксов
/stats - Статистика базы комиксов
```
