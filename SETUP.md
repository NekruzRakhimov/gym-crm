# Gym CRM — Инструкция по запуску

## Требования

| Инструмент | Версия |
|-----------|--------|
| Go | 1.22+ |
| Node.js | 18+ |
| PostgreSQL | 14+ |
| Git | любая |

---

## 1. Клонирование / копирование проекта

```bash
# Если проект на Git:
git clone <url-репозитория>
cd gym-crm

# Структура папок:
# gym-crm/
#   gym-crm-back/   — Go бэкенд
#   gym-crm-front/  — React фронтенд
```

---

## 2. База данных (PostgreSQL)

### Создать БД

```bash
psql -U postgres
```

```sql
CREATE DATABASE gym_crm;
\q
```

> Миграции применяются **автоматически** при старте сервера — вручную ничего запускать не нужно.

---

## 3. Бэкенд (Go)

### 3.1 Настроить переменные окружения

```bash
cd gym-crm-back
cp .env.example .env   # или создай .env вручную
```

Открой `.env` и заполни:

```env
# Строка подключения к PostgreSQL
DB_URL=postgres://postgres:ВАШ_ПАРОЛЬ@localhost:5432/gym_crm?sslmode=disable

# Секреты для JWT — придумай любые длинные строки
JWT_ACCESS_SECRET=измени-на-случайную-строку-32-символа
JWT_REFRESH_SECRET=измени-на-другую-случайную-строку

# IP этого компьютера в локальной сети (для настройки webhook на терминалах)
SERVER_IP=192.168.1.XXX

# Порт сервера
SERVER_PORT=8080

# Данные администратора по умолчанию (создаётся при первом запуске)
ADMIN_USERNAME=admin
ADMIN_PASSWORD=ваш-пароль

# Папка для хранения фото клиентов
UPLOADS_DIR=./uploads
```

> **Узнать свой IP**: `ip addr` (Linux) или `ipconfig` (Windows) или `ifconfig` (macOS)

### 3.2 Установить зависимости и запустить

```bash
cd gym-crm-back

# Скачать зависимости
go mod download

# Запустить сервер
go run cmd/server/main.go
```

При успешном запуске увидишь:
```
2026/xx/xx xx:xx:xx starting server on :8080
```

> При первом запуске автоматически:
> - Применяются все миграции (`migrations/*.sql`)
> - Создаётся admin-пользователь из `.env`

---

## 4. Фронтенд (React)

```bash
cd gym-crm-front

# Установить зависимости
npm install

# Запустить в режиме разработки
npm run dev
```

Фронтенд будет доступен по адресу: **http://localhost:5173**

> Все запросы к `/api`, `/ws`, `/uploads` автоматически проксируются на `localhost:8080`.

---

## 5. Вход в систему

Открой **http://localhost:5173** в браузере.

Войди с данными из `.env`:
- **Логин**: значение `ADMIN_USERNAME`
- **Пароль**: значение `ADMIN_PASSWORD`

---

## 6. Настройка терминалов Hikvision

После входа в админку:

1. Перейди в **Терминалы** → добавь каждый терминал (IP, порт, логин, пароль)
2. Для каждого терминала нажми **Setup Webhook** — терминал начнёт отправлять события на сервер
3. Нажми **Sync** — все клиенты синхронизируются на терминал

> Терминалы должны быть в **одной локальной сети** с сервером.

---

## 7. Запуск в фоне

### Вариант А — nohup (быстро, для тестирования)

```bash
# Собрать бинарник бэкенда
cd gym-crm-back
go build -o gym-crm-server cmd/server/main.go

# Запустить бэкенд в фоне
nohup ./gym-crm-server > server.log 2>&1 &

# Собрать фронтенд и запустить preview
cd ../gym-crm-front
npm run build
nohup npm run preview > frontend.log 2>&1 &
```

Остановить:
```bash
pkill -f gym-crm-server
pkill -f "vite preview"
```

Логи смотреть: `tail -f gym-crm-back/server.log`

---

### Вариант Б — systemd (рекомендуется для постоянной работы на Linux)

**Шаг 1.** Собери бэкенд:
```bash
cd gym-crm-back
go build -o /opt/gym-crm/server cmd/server/main.go
cp .env /opt/gym-crm/.env
cp -r migrations /opt/gym-crm/migrations
mkdir -p /opt/gym-crm/uploads
```

**Шаг 2.** Создай systemd-сервис:
```bash
sudo nano /etc/systemd/system/gym-crm.service
```

```ini
[Unit]
Description=Gym CRM Backend
After=network.target postgresql.service

[Service]
Type=simple
User=www-data
WorkingDirectory=/opt/gym-crm
EnvironmentFile=/opt/gym-crm/.env
ExecStart=/opt/gym-crm/server
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

**Шаг 3.** Включи и запусти:
```bash
sudo systemctl daemon-reload
sudo systemctl enable gym-crm
sudo systemctl start gym-crm

# Статус
sudo systemctl status gym-crm

# Логи
sudo journalctl -u gym-crm -f
```

> Фронтенд в продакшне раздаётся через nginx (см. раздел 8) — отдельный сервис для него не нужен.

---

## 8. Для продакшна (сборка фронтенда)

Если нужно запустить без `npm run dev`:

```bash
cd gym-crm-front
npm run build
```

Собранные файлы появятся в `gym-crm-front/dist/`.
Их нужно раздавать через nginx или любой статик-сервер.

Пример конфига nginx:

```nginx
server {
    listen 80;
    server_name ваш-домен-или-ip;

    # Фронтенд
    location / {
        root /path/to/gym-crm-front/dist;
        try_files $uri $uri/ /index.html;
    }

    # Бэкенд API
    location /api/ {
        proxy_pass http://localhost:8080;
    }

    location /ws {
        proxy_pass http://localhost:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
    }

    location /uploads/ {
        proxy_pass http://localhost:8080;
    }
}
```

Не забудь обновить CORS в бэкенде (`router/router.go`):
```go
AllowOrigins: []string{"http://ваш-домен-или-ip"},
```

---

## 8. Часто встречаемые проблемы

| Проблема | Решение |
|---------|---------|
| `connect db: ...` при старте | Проверь `DB_URL` в `.env`, убедись что PostgreSQL запущен |
| Фронтенд не открывается | Убедись что бэкенд запущен на порту 8080 |
| Терминалы не отправляют события | Проверь `SERVER_IP` в `.env` — должен быть IP в локальной сети, не `127.0.0.1` |
| `invalid credentials` при входе | Первый запуск уже был? Проверь `ADMIN_USERNAME`/`ADMIN_PASSWORD` в `.env` |
| Фото не синхронизируются | Убедись что папка `uploads/` существует и доступна для записи |
