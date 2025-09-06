#!/bin/bash
set -e

echo "Waiting for database to be ready..."
sleep 15

echo "Initializing Airflow database..."
airflow db init

echo "Creating admin user..."
airflow users create \
    --username admin \
    --firstname Admin \
    --lastname User \
    --role Admin \
    --email admin@example.com \
    --password admin

echo "Starting Airflow services..."
airflow webserver --port 8080 &
airflow scheduler
