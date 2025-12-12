# SonarQube 독립 실행 환경

SonarQube 코드 품질 분석만을 위한 경량화된 Docker Compose 환경입니다.

## 🚀 빠른 시작

### 1. 환경 시작
```bash
# Makefile 사용 (권장)
make sonar-up

# 또는 직접 스크립트 실행
./docker/scripts/sonar.sh up
```

### 2. SonarQube 접속
- URL: http://localhost:9000
- 기본 로그인: `admin` / `admin`
- 첫 로그인 시 비밀번호 변경 필요

### 3. 환경 중지
```bash
make sonar-down
```

## 📋 사용 가능한 명령어

| 명령어 | 설명 |
|--------|------|
| `make sonar-up` | SonarQube 환경 시작 |
| `make sonar-down` | SonarQube 환경 중지 |
| `make sonar-logs` | 로그 확인 |
| `make sonar-status` | 상태 확인 |
| `make sonar-restart` | 환경 재시작 |
| `make sonar-clean` | 데이터 완전 삭제 |

## 🔍 코드 분석 예시

### Go 서비스 분석
```bash
cd services/user-service

# 테스트 커버리지 생성
go test -coverprofile=coverage.out ./...

# SonarScanner 실행
sonar-scanner \
  -Dsonar.projectKey=wealist-user-service \
  -Dsonar.projectName="weAlist User Service" \
  -Dsonar.sources=. \
  -Dsonar.exclusions="**/*_test.go,**/vendor/**" \
  -Dsonar.go.coverage.reportPaths=coverage.out \
  -Dsonar.host.url=http://localhost:9000 \
  -Dsonar.token=YOUR_TOKEN_HERE
```

### Java 서비스 분석 (auth-service)
```bash
cd services/auth-service

# Maven 사용
mvn clean verify sonar:sonar \
  -Dsonar.projectKey=wealist-auth-service \
  -Dsonar.host.url=http://localhost:9000 \
  -Dsonar.token=YOUR_TOKEN_HERE
```

### React 프론트엔드 분석
```bash
cd services/frontend

# 테스트 커버리지 생성
npm test -- --coverage

# SonarScanner 실행
sonar-scanner \
  -Dsonar.projectKey=wealist-frontend \
  -Dsonar.projectName="weAlist Frontend" \
  -Dsonar.sources=src \
  -Dsonar.exclusions="**/*.test.ts,**/*.test.tsx,**/node_modules/**" \
  -Dsonar.typescript.lcov.reportPaths=coverage/lcov.info \
  -Dsonar.host.url=http://localhost:9000 \
  -Dsonar.token=YOUR_TOKEN_HERE
```

## 🏗️ 아키텍처

### 포함된 서비스
- **SonarQube**: 코드 품질 분석 (포트 9000)
- **PostgreSQL**: SonarQube 데이터베이스 (포트 5433)

### 네트워크 격리
- 독립적인 네트워크: `wealist-sonarqube-standalone-net`
- 기존 전체 환경과 충돌 없음

### 데이터 지속성
- 기존 볼륨 재사용으로 데이터 공유
- 환경 전환 시에도 분석 결과 유지

## ⚠️ 주의사항

1. **포트 충돌 방지**
   - PostgreSQL: 5433 포트 사용 (기존 5432와 구분)
   - SonarQube: 9000 포트 (기존과 동일)

2. **환경 독립성**
   - 기존 `make dev-up` 환경과 독립적으로 동작
   - 동시 실행 가능하지만 리소스 사용량 증가

3. **데이터 공유**
   - 볼륨을 공유하므로 분석 결과가 환경 간 유지됨
   - `make sonar-clean` 사용 시 모든 데이터 삭제됨

## 🛠️ 문제 해결

### SonarQube가 시작되지 않는 경우
```bash
# 로그 확인
make sonar-logs

# 상태 확인
make sonar-status

# 환경 재시작
make sonar-restart
```

### 포트 충돌 발생 시
```bash
# 기존 환경 중지
make dev-down

# 또는 포트 사용 중인 프로세스 확인
lsof -i :9000
lsof -i :5433
```

### 환경변수 오류 시
```bash
# 환경변수 파일 확인
cat docker/env/.env.dev

# 템플릿에서 재생성
cp docker/env/.env.dev.example docker/env/.env.dev
```

## 📚 추가 자료

- [SonarQube 공식 문서](https://docs.sonarqube.org/latest/)
- [SonarScanner 설치 가이드](https://docs.sonarqube.org/latest/analyzing-source-code/scanners/sonarscanner/)
- [프로젝트 SonarQube 가이드](../SONARQUBE_GUIDE.md)