# README #

1. install golang (version 1.9 or higher)
2. set GOPATH https://github.com/golang/go/wiki/SettingGOPATH
3. create structure GOPATH/src/bitbucket.org/sotavant/rabbitOxpa
4. cd to path from point 3
5. clone repo: "git clone git@bitbucket.org:sotavant/rabbitoxpa.git ." Dot is required
6. go get github.com/streadway/amqp
7. go get github.com/BurntSushi/toml
8. go get -u github.com/go-sql-driver/mysql
9. make copy of gefault config with name "config"
10. actualize config and sure that dir and files is exists
11. go build
12. for debug ./rabbitOxpa (for prod ./rabbitOxpa &). 

## Предупреждение падения
Желательно использовать сервис наподобие http://supervisord.org/, который следит за тем чтобы программа всегда была запущена

## Сохранение состояние при восстановлении работы диспетчера
Для того чтобы сбросить сохраненное состояние. Например в случае очистки очереди или если она другая (новая), нужно 
удалить файл "state

## БД логирование

реализовано логирование в базу количества файлов, при генерации которых возникают проблемы
В конфиге задать значения
- LogField - поле в БД
- DocGenError - сообщение, которые возвращает генератор при ошибках обработки файлов

## Запуск постСкрипта

- добавить в конфиг путь до скрипта
- скрипту передается три параметра: 
1) номер задания, 
2) если была ошибка при генерации архива, то 1 инача 0
3) кол-во ошибок при генерации документов (тоже кол-во что и базу пишется)

## Обновить на сервере
1) cd gopath/
2) git pull origin master
3) go build
4) sudo supervisorctl restart rabbitoxpa_docs

## Параметр для пути до архива организации
companyArchivePath = "/26/226" (не обязательный, если параметр не задан, то по-умолчанию архив будет создаватся в папке PathToResultZip)