# API Сервис "Учёт Расходов" (Go)

RESTful API сервис, разработанный на Go и Gin, для управления личными доходами и расходами. Пользователи могут отслеживать свои транзакции и прикреплять чеки, в то время как администраторы имеют доступ к общим данным и агрегированной статистике.

## Основные Возможности

**Для Пользователей:**
*   Безопасная регистрация и вход с использованием JWT-аутентификации.
*   Добавление, просмотр, обновление и удаление личных финансовых транзакций (доходы/расходы).
*   Категоризация транзакций и добавление описаний.
*   Загрузка и получение файлов-чеков (изображения/PDF) для транзакций.
*   Фильтрация личных транзакций по типу, категории и дате.

**Для Администраторов:**
*   Просмотр всех транзакций всех пользователей.
*   Фильтрация транзакций по пользователю, категории, типу и периоду дат.
*   Доступ к агрегированной финансовой статистике (общий доход/расход, разбивка по категориям и пользователям).
*   Экспорт данных о транзакциях в CSV-файлы.

## Технологический Стек

*   **Язык:** Go (1.20+)
*   **Фреймворк:** Gin
*   **База данных:** PostgreSQL (с использованием драйвера `pgx`)
*   **Аутентификация:** JWT (JSON Web Tokens)
*   **Хеширование паролей:** bcrypt
*   **Конфигурация окружения:** Файлы `.env`
*   **Миграции БД:** Автоматическое создание/обновление схемы при запуске приложения (Auto DDL)


## Предварительные Требования

*   Go 1.20+
*   Docker и Docker Compose
*   Git

## Запуск Проекта

1.  **Клонируйте репозиторий:**
    ```bash
    git clone <URL_вашего_репозитория>
    cd expense-tracker-go # или имя вашей папки
    ```

2.  **Создайте файл `.env`:**
    Скопируйте пример ниже или создайте свой файл `.env` в корне проекта:
    ```dotenv
    SERVER_PORT=8080
    DB_HOST=localhost
    DB_PORT=5432
    DB_USER=expense_user      # Должен совпадать с docker-compose.yml
    DB_PASSWORD=expense_pass  # Должен совпадать с docker-compose.yml
    DB_NAME=expense_tracker2  # Должен совпадать с docker-compose.yml
    DB_SSLMODE=disable
    JWT_SECRET_KEY=ваш_очень_надёжный_случайный_jwt_секретный_ключ
    JWT_EXPIRATION_HOURS=24
    UPLOADS_DIR=uploads
    # Для первоначальной настройки администратора (опционально, используйте один раз, затем удалите/закомментируйте)
    # INITIAL_ADMIN_PHONE=телефон_вашего_администратора
    ```
    **Важно:** Замените `ваш_очень_надёжный_случайный_jwt_секретный_ключ` на сильный, уникальный ключ.

3.  **Запустите базу данных PostgreSQL:**
    ```bash
    docker-compose up -d
    ```

4.  **Установите Go-зависимости:**
    ```bash
    go mod tidy
    ```

5.  **Запустите приложение:**
    ```bash
    go run cmd/server/main.go
    ```
    API будет доступен по адресу `http://localhost:8080` (или по порту, указанному в `SERVER_PORT`).

## Обзор API Эндпоинтов

Все эндпоинты имеют префикс `/api/v1`. Для защищённых маршрутов требуется заголовок `Authorization: Bearer <JWT>`.

*   **Аутентификация:**
    *   `POST /auth/register`
    *   `POST /auth/login`
*   **Транзакции пользователя (требуется аутентификация):**
    *   `POST /transactions`
    *   `GET /transactions` (поддерживает query-параметры `type`, `category`, `date`)
    *   `GET /transactions/{id}`
    *   `PUT /transactions/{id}`
    *   `DELETE /transactions/{id}`
    *   `POST /transactions/{id}/receipt` (multipart/form-data)
    *   `GET /transactions/{id}/receipt`
*   **Административные функции (требуется аутентификация как администратор):**
    *   `GET /admin/transactions` (поддерживает query-параметры `user_id`, `type`, `category`, `start_date`, `end_date`)
    *   `GET /admin/stats` (те же фильтры)
    *   `GET /admin/transactions/export/csv` (те же фильтры)

