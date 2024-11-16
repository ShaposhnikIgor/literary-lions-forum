#!/bin/bash

# Получение текущей версии Go
GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')

# Проверка установки Go
if [ -z "$GO_VERSION" ]; then
  echo "Go is not installed on this system."
  exit 1
fi

# Указание базового образа (по умолчанию, например, `ubuntu`)
BASE_IMAGE=${1:-ubuntu:latest}

echo "Building Docker image with Go version $GO_VERSION and base image $BASE_IMAGE..."

# Сборка Docker-образа с указанием версии Go и базового образа
docker build --build-arg GO_VERSION=$GO_VERSION --build-arg BASE_IMAGE=$BASE_IMAGE -t literary-jo:latest .

# Удаление старого контейнера, если существует
docker rm -f literary-jo-container 2>/dev/null

# Запуск контейнера
docker run --name literary-jo-container -d -p 8080:8080 literary-jo:latest

echo "Container literary-jo-container is running with Go version $GO_VERSION and base image $BASE_IMAGE"
