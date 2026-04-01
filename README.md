# mtg-multi

Fork of [9seconds/mtg](https://github.com/9seconds/mtg) with multi-secret support and per-user stats.

[English](#whats-different) | [Русский](#чем-отличается)

---

## What's different

**Multiple secrets.** Upstream mtg allows only one secret per instance. mtg-multi lets you define named secrets in the config — one per user. All secrets must share the same hostname.

```toml
[secrets]
alice = "ee367a189aee18fa31c190054efd4a8e9573746f726167652e676f6f676c65617069732e636f6d"
bob   = "ee0123456789abcdef0123456789abcd9573746f726167652e676f6f676c65617069732e636f6d"
```

**Stats API.** A lightweight HTTP endpoint that shows live per-user traffic.

```toml
api-bind-to = "127.0.0.1:9090"
```

```
GET /stats
```

```json
{
  "started_at": "2026-03-29T10:30:00Z",
  "uptime_seconds": 3600,
  "total_connections": 15,
  "users": {
    "alice": {
      "connections": 8,
      "bytes_in": 1048576,
      "bytes_out": 2097152,
      "last_seen": "2026-03-29T11:25:30Z"
    }
  }
}
```

**Connection throttling.** Automatic per-user connection limits to protect the server from overload. A background goroutine recomputes caps every few seconds using a fair-share algorithm: small users keep their connections, remaining budget is split equally among heavy consumers. New connections from over-cap users are rejected; existing connections are not killed.

```toml
[throttle]
max-connections = 5000
check-interval = "5s"
```

Example: limit = 100, users A=1, B=1, C=90, D=110.
A and B stay at 1. Remaining budget 98 is split: C and D are capped at 49 each.

Throttle state is exposed via the Stats API:

```json
{
  "throttle": {
    "active": true,
    "limit": 5000,
    "caps": { "heavy-user": 2450 }
  }
}
```

**Public IP override.** Useful when auto-detection via ifconfig.co is unavailable.

```toml
public-ipv4 = "1.2.3.4"
public-ipv6 = "2001:db8::1"
```

Everything else — domain fronting, doppelganger, proxy chaining, blocklists, metrics — works exactly as in upstream. See the [upstream README](https://github.com/9seconds/mtg) for details.

## Quick start

Download a binary from [Releases](https://github.com/dolonet/mtg-multi/releases) or build from source:

```console
git clone https://github.com/dolonet/mtg-multi.git
cd mtg-multi
mise install && mise tasks run build
```

Generate secrets:

```console
mtg-multi generate-secret --hex storage.googleapis.com
```

Minimal config:

```toml
bind-to = "0.0.0.0:443"
api-bind-to = "127.0.0.1:9090"

[throttle]
max-connections = 5000

# [secrets] must be the last section in the global scope —
# in TOML, all keys after a [section] become part of that table.
[secrets]
alice = "ee..."
bob   = "ee..."
```

Run:

```console
mtg-multi run /etc/mtg/config.toml
```

See [example.config.toml](example.config.toml) for all available options.

---

## Чем отличается

**Несколько секретов.** В оригинальном mtg — один секрет на инстанс. mtg-multi позволяет задать именованные секреты в конфиге, по одному на пользователя. Все секреты должны использовать один и тот же hostname.

```toml
[secrets]
alice = "ee367a189aee18fa31c190054efd4a8e9573746f726167652e676f6f676c65617069732e636f6d"
bob   = "ee0123456789abcdef0123456789abcd9573746f726167652e676f6f676c65617069732e636f6d"
```

**Stats API.** HTTP-эндпоинт с live-статистикой трафика по пользователям.

```toml
api-bind-to = "127.0.0.1:9090"
```

```
GET /stats
```

```json
{
  "started_at": "2026-03-29T10:30:00Z",
  "uptime_seconds": 3600,
  "total_connections": 15,
  "users": {
    "alice": {
      "connections": 8,
      "bytes_in": 1048576,
      "bytes_out": 2097152,
      "last_seen": "2026-03-29T11:25:30Z"
    }
  }
}
```

**Троттлинг подключений.** Автоматические per-user лимиты для защиты сервера от перегрузки. Фоновая горутина каждые несколько секунд пересчитывает капы по алгоритму fair-share: маленькие пользователи сохраняют свои подключения, оставшийся бюджет делится поровну между крупными потребителями. Новые подключения сверх капа отклоняются; существующие не разрываются.

```toml
[throttle]
max-connections = 5000
check-interval = "5s"
```

Пример: лимит = 100, пользователи A=1, B=1, C=90, D=110.
A и B остаются на 1. Оставшийся бюджет 98 делится: C и D получают кап 49.

Состояние троттлинга доступно через Stats API:

```json
{
  "throttle": {
    "active": true,
    "limit": 5000,
    "caps": { "heavy-user": 2450 }
  }
}
```

**Ручное указание публичного IP.** Для случаев, когда ifconfig.co недоступен с сервера.

```toml
public-ipv4 = "1.2.3.4"
public-ipv6 = "2001:db8::1"
```

Всё остальное — domain fronting, doppelganger, цепочки прокси, блоклисты, метрики — работает как в оригинале. Подробности в [README upstream](https://github.com/9seconds/mtg).

## Быстрый старт

Скачайте бинарник из [Releases](https://github.com/dolonet/mtg-multi/releases) или соберите из исходников:

```console
git clone https://github.com/dolonet/mtg-multi.git
cd mtg-multi
mise install && mise tasks run build
```

Генерация секрета:

```console
mtg-multi generate-secret --hex storage.googleapis.com
```

Минимальный конфиг:

```toml
bind-to = "0.0.0.0:443"
api-bind-to = "127.0.0.1:9090"

[throttle]
max-connections = 5000

# [secrets] должен быть последней секцией в глобальном scope —
# в TOML все ключи после [section] становятся частью этой таблицы.
[secrets]
alice = "ee..."
bob   = "ee..."
```

Запуск:

```console
mtg-multi run /etc/mtg/config.toml
```

Все доступные опции — в [example.config.toml](example.config.toml).
