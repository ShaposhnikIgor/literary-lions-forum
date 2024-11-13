#!/bin/bash

# Получите текущую версию Go, установленную на вашей системе
GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')

# Если Go не установлен, вывести сообщение и выйти
if [ -z "$GO_VERSION" ]; then
  echo "Go is not installed on this system."
  exit 1
fi

echo "Building Docker image with Go version $GO_VERSION..."

# Постройте Docker-образ, передавая версию Go как аргумент
docker build --build-arg GO_VERSION=$GO_VERSION -t literary-jo:latest .

# Удалите старый контейнер с тем же именем, если он существует
docker rm -f literary-jo-container 2>/dev/null

# Запустите контейнер с именем literary-jo-container
docker run --name literary-jo-container -d -p 8080:8080 literary-jo:latest

echo "Container literary-jo-container is running with Go version $GO_VERSION"
