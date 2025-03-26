![Go Report](https://goreportcard.com/badge/github.com/mgdevgo/file-service)
![Repository Top Language](https://img.shields.io/github/languages/top/mgdevgo/file-service)
![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/mgdevgo/file-service)
![Github Repository Size](https://img.shields.io/github/repo-size/mgdevgo/file-service)
![License](https://img.shields.io/badge/license-MIT-green)
![GitHub last commit](https://img.shields.io/github/last-commit/mgdevgo/file-service)
![GitHub contributors](https://img.shields.io/github/contributors/mgdevgo/file-service)

# file-service

## How to run

Create `.env` file

```shell
DATABASE_URL=postgres://postgres:postgres@localhost:5432/postgres
FILES_UPLOAD_PATH=./uploads
```

Run locally

```shell
make run
```

Run in Docker

```shell
docker-compose up --build
```

## Requirements

Необходимо написать сервис на Golang работающий по gRPC.

Сервис должен:
1. Принимать бинарные файлы (изображения) от клиента и сохранять их на жесткий диск.
2. Иметь возможность просмотра списка всех загруженных файлов в формате: `Имя файла | Дата создания | Дата обновления`
4. Отдавать файлы клиенту.
5. Ограничивать количество одновременных подключений с клиента:
   - на загрузку/скачивание файлов - 10 конкурентных запросов
   - на просмотр списка файлов - 100 конкурентных запросов

## Architecture overview

![Architecture overview](./docs/architecture.png)

## Service description

Метаданные файлов сохраняются в базе данных PostgreSQL. 

Метаданные включают уникальный идентификатор файла (UUID), имя файла, хэш содержимого, дату создания и дату обновления.

Файлы сохраняются на жесткий диск в директорию, указанную в переменной окружения `FILES_UPLOAD_PATH`. Для организации хранения файлов используется подход `content-addressable storage`. 

## What can be improved

- Добавить кэширование файлов
- Добавить возможность скачивать файлы по частям


