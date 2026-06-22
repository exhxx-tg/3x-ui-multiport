# Backup API

## Create Backup

```http
POST /panel/api/backup/create
```

Creates an encrypted backup of the database and configuration.

## List Backups

```http
GET /panel/api/backup/list
```

## Restore Backup

```http
POST /panel/api/backup/restore/:id
```

Restores from a specific backup.

## Delete Backup

```http
DELETE /panel/api/backup/:id
```

## Telegram Bot Backup

```http
POST /panel/api/backuptotgbot
```

Sends a fresh database backup to configured Telegram chats.

## Download Database

```http
GET /panel/api/server/getDb
```

Streams the SQLite database file as an attachment.

## Import Database

```http
POST /panel/api/server/importDB
```

Restores from an uploaded SQLite file (multipart form).
