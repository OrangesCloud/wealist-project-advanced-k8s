#!/bin/bash
set -e

# SonarQube 전용 PostgreSQL 초기화 스크립트
# SonarQube 데이터베이스와 사용자만 생성합니다

echo "🚀 SonarQube 데이터베이스 초기화 시작..."

# SonarQube Database
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" <<-EOSQL
    CREATE DATABASE ${SONARQUBE_DB_NAME};
    CREATE USER ${SONARQUBE_DB_USER} WITH PASSWORD '${SONARQUBE_DB_PASSWORD}';
    GRANT ALL PRIVILEGES ON DATABASE ${SONARQUBE_DB_NAME} TO ${SONARQUBE_DB_USER};
    \c ${SONARQUBE_DB_NAME}
    GRANT ALL ON SCHEMA public TO ${SONARQUBE_DB_USER};
EOSQL

echo "✅ SonarQube 데이터베이스 생성 완료: ${SONARQUBE_DB_NAME}"
echo "🎉 SonarQube 데이터베이스 초기화 완료!"
echo "📋 생성된 데이터베이스:"
echo "   - ${SONARQUBE_DB_NAME} (SonarQube - Code Quality)"