# go-musthave-shortener-tpl

Шаблон репозитория для трека «Сервис сокращения URL».

## Начало работы

1. Склонируйте репозиторий в любую подходящую директорию на вашем компьютере.
2. В корне репозитория выполните команду `go mod init <name>` (где `<name>` — адрес вашего репозитория на GitHub без префикса `https://`) для создания модуля.

## Обновление шаблона

Чтобы иметь возможность получать обновления автотестов и других частей шаблона, выполните команду:

```
git remote add -m v2 template https://github.com/Yandex-Practicum/go-musthave-shortener-tpl.git
```

Для обновления кода автотестов выполните команду:

```
git fetch template && git checkout template/v2 .github
```

Затем добавьте полученные изменения в свой репозиторий.

## Запуск автотестов

Для успешного запуска автотестов называйте ветки `iter<number>`, где `<number>` — порядковый номер инкремента. Например, в ветке с названием `iter4` запустятся автотесты для инкрементов с первого по четвёртый.

При мёрже ветки с инкрементом в основную ветку `main` будут запускаться все автотесты.

Подробнее про локальный и автоматический запуск читайте в [README автотестов](https://github.com/Yandex-Practicum/go-autotests).

## Структура проекта

Приведённая в этом репозитории структура проекта является рекомендуемой, но не обязательной.

Это лишь пример организации кода, который поможет вам в реализации сервиса.

При необходимости можно вносить изменения в структуру проекта, использовать любые библиотеки и предпочитаемые структурные паттерны организации кода приложения, например:
- **DDD** (Domain-Driven Design)
- **Clean Architecture**
- **Hexagonal Architecture**
- **Layered Architecture**


## Документация
/swagger/index.html

## Запуск кастомного анализатора
Команды для запуска:
- MacOS - `make check-darwin PACKAGE="<path>"` или всего проекта `make check-darwin`
- Linux - `make check-linux PACKAGE="<path>"` или всего проекта `make check-linux`

## Запуск утилиты, генерирующая функции очистки для структур
Команда для запуска - `make run-reset`

## Сборка приложения
Команда для сборки `go build -ldflags="-X main.buildVersion=<VERSION> -X 'main.buildDate=<DATETIME>' -X main.buildCommit=<COMMIT>" -o ./shortener ./cmd/shortener`
Дефолтный значения `Build version`, `Build date`, `Build commit` - `N/A`

Через Makefile
- MacOS - `make build-darwin VERSION=<any> COMMIT=<any>`
- Linux - `make build-linux VERSION=<any> COMMIT=<any>`

Дефолтный значения `Build version`, `Build commit` - `N/A`, `Build date` - текущее время (utc+3 hourse) в формате - `'%Y/%m/%d-%H:%M:%S` 
