# go-musthave-shortener-tpl

## Сборка приложения
Команда для сборки приложения

`go build -ldflags="-X main.buildVersion=<VERSION> -X 'main.buildDate=<DATETIME>' -X main.buildCommit=<COMMIT>" -o ./<binary_file> ./cmd/shortener`

Дефолтный значения `Build version`, `Build date`, `Build commit` - `N/A`

Через Makefile
- MacOS - `make build-darwin VERSION=<any> COMMIT=<any>`
- Linux - `make build-linux VERSION=<any> COMMIT=<any>`

Дефолтный значения `Build version`, `Build commit` - `N/A`, `Build date` - текущее время (utc + 3 часа) в формате - `'%Y/%m/%d-%H:%M:%S`


## Запуск приложения
Доступные для запуска флаги командной строки и переменные окружения (флаг/переменная окружения)
- `-a`/`SERVER_ADDRESS` - адрес сервера, дефолтное значение - `localhost:8080`
- `-b`/`BASE_URL` - базовый адрес результирующего сокращённого URL, дефолное значение - `http://localhost:8080`
- `-f`/`FILE_STORAGE_PATH` - путь до файла, куда сохраняются данные в формате JSON
- `-debug`/`DEBUG_LEVEL` - уровень логирования сервиса
- `-d`/`DATABASE_DSN` - адрес подключения к СУБД (postgres)
- `-audit-file`/`AUDIT_FILE` - путь к файлу-приёмнику, в который сохраняются логи аудита
- `-audit-url`/`AUDIT_URL` - полный URL удаленного сервера-приёмника, куда отправляются логи аудита
- `-s`/`ENABLE_HTTPS` - запуск приложения на HTTPS сервере
- `-cf`/`CERT_FILE` - путь до публичных TLS сертификатов
- `-kf`/`KEY_FILE` - путь до приватного ключа TLS сертификатов
- `-hwl`/`HOST_WHITE_LIST` - список доступных адресов для HTTPS сервера (строка из адресов, разделенные запятой без пробела, например - `test1.com,test2.com,test3.com`)
- `-m` - запуск приложения в режиме разработки (dev) или продакшена (prod)
- `-с`/`JSON_CONFIG` - путь до json-файла с конфигом приложения

Команда для запуска сервиса
- без предварительной сборки - `./<binary_file> <any flags...>`

где `binary_file` - название предварительно собранной бинарного файла

## Документация
Эндпоинт swagger - `<SERVER_ADDRESS>/swagger/index.html`

## Кастомный анализатор кода
Команды для запуска:
- MacOS - `make check-darwin PACKAGE="<path>"` или всего проекта `make check-darwin`
- Linux - `make check-linux PACKAGE="<path>"` или всего проекта `make check-linux`

## Утилита, генерирующая функции очитстки для стуктур
У всех структур с комментарием `// generate:reset` создается метод Reset(), который задаёт дефолтные (нулевые) значения в зависимости от типа данных поля

Команда для запуска
- `go run ./cmd/reset -dir=<path>`

Через Makefile
- `make run-reset dir=<path>`

где необязательные флаг `-dir` и параметр `dir` - относительный путь от корневой директории проекта (дефолтное значение - корневая директория `./...`)
