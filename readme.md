
# Тестовое задание, backend

Разработать HTTP сервис на языке Go слушающий порт 4567, который на GET запрос вернет HTML страничку, на которой будет виден курсор каждого пользователя, у которого эта страничка открыта. Курсоры других пользователей можно показать простыми кружками разных цветов.
Ниже представлен клиентский код который можно менять по своему желанию, также этот код может содержать ошибки, если они есть – то их нужно поправить.

Веб-страничка  слушает событие onmousemove и отправлять значения на бекенд, который в свою очередь должен отправлять эту информацию всем остальным пользователям.
(HTML страница должна содержать несколько строк JavaScript кода, которые будут осуществлять подключение обратно к серверу используя WebSockets или EventSource для обменом данных).
Если один из пользователей закроет вкладку (соединение прервется), курсор этого пользователя должен пропасть всех остальных пользователей.
Каждая открытая вкладка считается за отдельного пользователя.

Допускается использование сторонних библиотек для организации работы WebSockets или EventSource со стороны Go.

## Запуск

```sh
cd src && go run *.go
```

Откройте в бораузере http://localhost:4567/ В нескольких вкладках# cursor-viewer-example