## PullRequest-Manager

###  Cтарт

#### Запуск в docker
```bash
make up
```
Сервис, postgresql и linter запускаются автоматически. Сервис будет доступен на `http://localhost:8080`, база — на `localhost:5433`.

Остановка:
```bash
make down
```

- `make up` — запуск сервиса и зависимостей в Docker
- `make down` — остановка и удаление контейнеров
- `make test` — запуск интеграционных тестов
- `make lint` — запуск линтера

### Токены
Для всех методов, кроме (`/team/get`, `/users/getReview`) требуется заголовок `Authorization: Bearer <admin-secret>`.

###  Эндпоинты
- `POST /team/add` — создание команды и участников
- `GET /team/get?team_name=...` — просмотр состава команды
- `POST /users/setIsActive` — изменение активности пользователя
- `POST /pullRequest/create` — создание PR с автоматическим назначением ревьюверов
- `POST /pullRequest/reassign` — переназначение ревьювера
- `POST /pullRequest/merge` — установка статуса `MERGED`
- `GET /users/getReview?user_id=...` — список PR пользователя
- `POST /team/deactivate` — деактивация и переприсвоение ревьюверов
- `GET /stats` — статистика

### Тестирование

#### unit-tests
```bash
go test ./internal/usecase/...
```

#### Интеграционные тесты
```bash
go test ./tests/integration/...
```

### Описание
#### Слои

1. **entities** — модели и ошибки:

2. **usecase** — бизнес-логика:
   - Создание PR, переназначение ревьюверов
   - Валидация данных
   - Логика случайного выбора ревьюверов
   - Обработка массовой деактивации с перераспределением PR

3. **repository** — интерфейсы доступа к данным:

4. **handler** — HTTP слой:
   - Маршрутизация (chi)
   - Аутентификация через Bearer токены
   - Преобразование HTTP запросов/ответов
   - Обработка ошибок и логирование

5. **infrastructure/postgres** — реализация репозиториев:
   - Реализует интерфейс `repository.Repository`

6. **db/migrations/postgresql** — миграции

