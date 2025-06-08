#!/bin/sh

# Устанавливаем bash-like опции: выходить при ошибке и при обращении к неустановленной переменной
set -e
set -u

# Проверяем, существует ли ключ сервисного аккаунта
if [ ! -f "key.json" ]; then
    echo "Ошибка: Файл key.json не найден!"
    exit 1
fi

# Проверяем, установлены ли переменные окружения для кластера
if [ -z "${K8S_CLUSTER_NAME}" ] || [ -z "${K8S_CLUSTER_ZONE}" ]; then
  echo "Ошибка: Переменные окружения K8S_CLUSTER_NAME и K8S_CLUSTER_ZONE должны быть установлены."
  exit 1
fi

echo "Аутентификация с помощью сервисного аккаунта..."
yc config set-service-account-key key.json

echo "Получение учетных данных для кластера Kubernetes: ${K8S_CLUSTER_NAME}..."
yc container cluster get-credentials "${K8S_CLUSTER_NAME}" --internal

echo "Конфигурация kubectl завершена."

# Запускаем основное приложение
echo "Запуск основного приложения..."
exec ./main

