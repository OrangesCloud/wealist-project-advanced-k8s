# SonarQube 통합 가이드

## 🎉 SonarQube 독립 환경이 구축되었습니다!

**SonarQube 10.3 Community Edition**이 독립적인 Docker 환경으로 구성되어 코드 품질 및 보안 분석을 제공합니다.

---

## 📊 개요

### SonarQube란?

SonarQube는 코드 품질 및 보안을 지속적으로 검사하는 오픈소스 플랫폼입니다.

**주요 기능**:
- 🐛 **버그 탐지**: 잠재적 버그와 코드 스멜 감지
- 🔒 **보안 취약점**: OWASP Top 10, CWE 기반 보안 이슈 발견
- 📏 **코드 커버리지**: 테스트 커버리지 추적
- 📈 **기술 부채**: 코드 개선에 필요한 시간 추정
- 🎯 **Quality Gates**: 코드 품질 기준 설정 및 자동 검증

### 아키텍처

```
┌─────────────────┐
│   개발자들       │
└────────┬────────┘
         │ (1) 코드 푸시
         ↓
┌─────────────────┐
│   GitHub/Git    │
└────────┬────────┘
         │ (2) 분석 트리거
         ↓
┌─────────────────┐      ┌──────────────┐
│  SonarScanner   │ ───→ │  SonarQube   │
│  (CI/CD or CLI) │      │  서버        │
└─────────────────┘      └──────┬───────┘
                                │ (3) 결과 저장
                                ↓
                         ┌──────────────┐
                         │  PostgreSQL  │
                         │  데이터베이스 │
                         └──────────────┘
```

---

## 🚀 빠른 시작

### 1. 서비스 시작

```bash
# SonarQube 독립 환경 시작
make sonar-up

# 또는 스크립트 직접 실행
./docker/scripts/sonar.sh up
```

**SonarQube 시작 시간**: 약 60-90초 (첫 시작 시 더 오래 걸릴 수 있음)

### 2. SonarQube 접속

```bash
# 브라우저에서 접속
open http://localhost:9000
```

**기본 로그인 정보**:
- 사용자명: `admin`
- 비밀번호: `admin`

**⚠️ 첫 로그인 시 비밀번호 변경 필수**

### 3. 상태 확인

```bash
# SonarQube 준비 상태 확인
make sonar-status

# 또는 API로 직접 확인
curl http://localhost:9000/api/system/status

# 예상 응답: {"status":"UP"}
```

---

## 🔧 설정

### 환경 변수

```bash
# docker/env/.env.dev
SONARQUBE_PORT=9000
SONARQUBE_DB_NAME=wealist_sonarqube_db
SONARQUBE_DB_USER=sonarqube_service
SONARQUBE_DB_PASSWORD=sonarqube_service_password
```

### 데이터베이스

SonarQube는 데이터 저장을 위해 PostgreSQL을 사용합니다:
- **데이터베이스**: `wealist_sonarqube_db`
- **사용자**: `sonarqube_service`
- **자동 생성**: `docker/init/postgres/init-sonarqube-db.sh`에 의해

### 볼륨

```yaml
volumes:
  sonarqube-data:       # 분석 결과, 설정
  sonarqube-extensions: # 플러그인
  sonarqube-logs:       # 애플리케이션 로그
```

**데이터 지속성**: 모든 데이터는 컨테이너 재시작 후에도 유지됩니다.

---

## �  토큰 생성 방법

SonarQube에서 코드 분석을 위해서는 **인증 토큰**이 필요합니다. 토큰을 생성하는 방법을 단계별로 설명합니다.

### 1단계: SonarQube 웹 UI 접속

```bash
# 브라우저에서 SonarQube 접속
open http://localhost:9000
```

### 2단계: 로그인

- **사용자명**: `admin`
- **비밀번호**: `admin` (첫 로그인 시)
- 첫 로그인 후 새 비밀번호로 변경 필수

### 3단계: 토큰 생성 (UI 방법)

1. **우상단 프로필 아이콘** 클릭 → **My Account** 선택
2. **Security** 탭 클릭
3. **Generate Tokens** 섹션에서:
   - **Token Name**: `wealist-analysis-token` (또는 원하는 이름)
   - **Type**: `Global Analysis Token` 선택
   - **Expires in**: `No expiration` (또는 원하는 기간)
4. **Generate** 버튼 클릭
5. **생성된 토큰을 복사하여 안전한 곳에 저장** ⚠️ 한 번만 표시됩니다!

### 4단계: 토큰 사용

생성된 토큰을 `sonar-project.properties` 파일에 입력:

```bash
# 예시 토큰 (실제로는 더 긴 문자열)
sonar.token=squ_1a2b3c4d5e6f7g8h9i0j1k2l3m4n5o6p7q8r9s0t
```

### API를 통한 토큰 생성 (고급)

```bash
# 새 비밀번호로 로그인 후 토큰 생성
curl -X POST -u admin:새로운비밀번호 \
  "http://localhost:9000/api/user_tokens/generate" \
  -d "name=wealist-analysis-token&type=GLOBAL_ANALYSIS_TOKEN"

# 응답 예시:
# {"login":"admin","name":"wealist-analysis-token","token":"squ_1a2b3c4d5e6f7g8h9i0j1k2l3m4n5o6p7q8r9s0t"}
```

### 토큰 보안 주의사항

- 🔒 **토큰을 안전하게 보관**하세요 (비밀번호와 동일하게 취급)
- 📝 **Git에 커밋하지 마세요** (`.gitignore`에 `sonar-project.properties` 추가 권장)
- 🔄 **정기적으로 토큰을 갱신**하세요
- 🗑️ **사용하지 않는 토큰은 삭제**하세요

### 빠른 토큰 생성 가이드

**1. 브라우저에서 접속**
```bash
open http://localhost:9000
```

**2. 로그인**
- ID: `admin`, PW: `admin` → 새 비밀번호 설정

**3. 토큰 생성**
- 우상단 **A** (Admin) 아이콘 → **My Account**
- **Security** 탭 → **Generate Tokens**
- Name: `my-token`, Type: `Global Analysis Token`
- **Generate** 클릭 → **토큰 복사**

**4. 토큰 사용**
```bash
# sonar-project.properties에 붙여넣기
sonar.token=복사한_토큰_여기에_붙여넣기
```

💡 **토큰 예시**: `squ_a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6q7r8s9t0` (실제로는 더 김)

---

## 📦 프로젝트 설정

### 프로젝트 생성

#### 방법 1: 수동 설정 (UI)

1. **이동**: http://localhost:9000 → Projects → Create Project
2. **프로젝트 키**: 예: `wealist-user-service`
3. **표시 이름**: `weAlist User Service`
4. **메인 브랜치**: `main`
5. **토큰 생성**:
   - 토큰 이름: `user-service-token`
   - 유형: Project Analysis Token
   - 토큰을 복사하여 저장

#### 방법 2: API (자동화)

```bash
# API를 통한 프로젝트 생성
curl -X POST -u admin:새로운-비밀번호 \
  "http://localhost:9000/api/projects/create" \
  -d "name=weAlist User Service&project=wealist-user-service"

# 토큰 생성
curl -X POST -u admin:새로운-비밀번호 \
  "http://localhost:9000/api/user_tokens/generate" \
  -d "name=user-service-token&projectKey=wealist-user-service"
```

### 권장 프로젝트 구성

서비스별로 하나의 프로젝트를 생성하세요:
- `wealist-auth-service` (Java/Spring Boot)
- `wealist-user-service` (Go)
- `wealist-board-service` (Go)
- `wealist-chat-service` (Go)
- `wealist-noti-service` (Go)
- `wealist-storage-service` (Go)
- `wealist-frontend` (React/TypeScript)

---

## 🔍 코드 분석

### 방법 1: SonarScanner CLI (로컬 개발 권장)

#### SonarScanner 설치

```bash
# macOS
brew install sonar-scanner

# Linux (수동 설치)
wget https://binaries.sonarsource.com/Distribution/sonar-scanner-cli/sonar-scanner-cli-5.0.1.3006-linux.zip
unzip sonar-scanner-cli-5.0.1.3006-linux.zip
export PATH=$PATH:/path/to/sonar-scanner/bin
```

#### Go 서비스 분석

```bash
cd services/user-service

# sonar-project.properties 파일 생성
cat > sonar-project.properties <<EOF
sonar.projectKey=wealist-user-service
sonar.projectName=weAlist User Service
sonar.projectVersion=1.0
sonar.sources=.
sonar.exclusions=**/*_test.go,**/vendor/**,**/migrations/**
sonar.go.coverage.reportPaths=coverage.out
sonar.host.url=http://localhost:9000
sonar.token=squ_여기에_실제_생성한_토큰_붙여넣기
EOF

# 커버리지와 함께 테스트 실행
go test -coverprofile=coverage.out ./...

# SonarScanner 실행
sonar-scanner
```

**⚠️ 중요**: `sonar.token=` 뒤에 위에서 생성한 실제 토큰을 붙여넣으세요!

#### Java 서비스 분석 (auth-service)

```bash
cd services/auth-service

# Maven
mvn clean verify sonar:sonar \
  -Dsonar.projectKey=wealist-auth-service \
  -Dsonar.projectName="weAlist Auth Service" \
  -Dsonar.host.url=http://localhost:9000 \
  -Dsonar.token=여기에_토큰_입력

# 또는 Gradle
./gradlew sonar \
  -Dsonar.projectKey=wealist-auth-service \
  -Dsonar.host.url=http://localhost:9000 \
  -Dsonar.token=여기에_토큰_입력
```

#### 프론트엔드 분석 (React/TypeScript)

```bash
cd services/frontend

# sonar-project.properties 파일 생성
cat > sonar-project.properties <<EOF
sonar.projectKey=wealist-frontend
sonar.projectName=weAlist Frontend
sonar.projectVersion=1.0
sonar.sources=src
sonar.exclusions=**/*.test.ts,**/*.test.tsx,**/node_modules/**,**/dist/**
sonar.typescript.lcov.reportPaths=coverage/lcov.info
sonar.host.url=http://localhost:9000
sonar.token=여기에_토큰_입력
EOF

# 커버리지와 함께 테스트 실행
npm test -- --coverage

# SonarScanner 실행
sonar-scanner
```

### 방법 2: GitHub Actions (CI/CD)

`.github/workflows/sonarqube.yml` 파일을 생성하세요:

```yaml
name: SonarQube 분석

on:
  push:
    branches: [main, develop]
  pull_request:
    branches: [main, develop]

jobs:
  sonarqube:
    runs-on: ubuntu-latest

    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0  # 더 나은 분석을 위한 전체 히스토리

      - name: Go 설정
        uses: actions/setup-go@v4
        with:
          go-version: '1.24'

      - name: 커버리지와 함께 테스트 실행
        run: |
          cd services/user-service
          go test -coverprofile=coverage.out ./...

      - name: SonarQube 스캔
        uses: sonarsource/sonarqube-scan-action@master
        env:
          SONAR_TOKEN: ${{ secrets.SONAR_TOKEN }}
          SONAR_HOST_URL: http://your-sonarqube-server:9000
        with:
          projectBaseDir: services/user-service
```

---

## 📈 품질 게이트

### 기본 품질 게이트

SonarQube는 기본 품질 게이트를 제공합니다:
- **버그**: 새로운 버그 0개
- **취약점**: 새로운 취약점 0개
- **보안 핫스팟**: 100% 검토 완료
- **코드 스멜**: 새로운 기술 부채 비율 ≤ 3%
- **커버리지**: 새 코드에서 ≥ 80%
- **중복**: 새 코드에서 ≤ 3%

### 사용자 정의 품질 게이트 (권장)

1. **이동**: Quality Gates → Create
2. **이름**: `weAlist Standard`
3. **조건**:
   - 새 코드 커버리지 ≥ 70%
   - 새 코드 중복 라인 ≤ 3%
   - 새 코드 유지보수성 등급 ≥ A
   - 새 코드 신뢰성 등급 ≥ A
   - 새 코드 보안 등급 ≥ A
4. **기본값으로 설정**: Actions → Set as Default

---

## 🔌 IDE 통합

### VS Code

**SonarLint** 확장 프로그램을 설치하세요:
```bash
code --install-extension SonarSource.sonarlint-vscode
```

`.vscode/settings.json` 설정:
```json
{
  "sonarlint.connectedMode.servers": [
    {
      "serverId": "wealist-local",
      "serverUrl": "http://localhost:9000",
      "token": "여기에_토큰_입력"
    }
  ],
  "sonarlint.connectedMode.project": {
    "serverId": "wealist-local",
    "projectKey": "wealist-user-service"
  }
}
```

### IntelliJ IDEA / GoLand

1. **플러그인 설치**: Settings → Plugins → SonarLint
2. **Connected Mode 설정**:
   - Settings → Tools → SonarLint → Connected Mode
   - **Add Connection** 클릭
   - Connection Name: `wealist-local`
   - Server URL: `http://localhost:9000`
   - Authentication: **Token** 선택
   - Token: 위에서 생성한 토큰 입력
   - **Test Connection** 클릭하여 연결 확인
3. **프로젝트 바인딩**:
   - Project Key 선택 (예: `wealist-board-service`)
   - **Bind** 클릭

### 🚨 IntelliJ 연결 문제 해결

**"Insufficient privileges" 오류 시**:
1. SonarQube에서 새 토큰 생성 (Global Analysis Token)
2. IntelliJ에서 기존 연결 삭제 후 새 토큰으로 재연결
3. 프로젝트가 SonarQube에 존재하는지 확인

**토큰 생성 명령어**:
```bash
curl -X POST -u admin:비밀번호 \
  "http://localhost:9000/api/user_tokens/generate" \
  -d "name=intellij-token&type=GLOBAL_ANALYSIS_TOKEN"
```

---

## 📊 모니터링 (Prometheus 통합)

SonarQube 메트릭은 Prometheus에 의해 자동으로 수집됩니다:

```yaml
# docker/monitoring/prometheus/prometheus.yml
- job_name: 'sonarqube'
  static_configs:
    - targets: ['sonarqube:9000']
  metrics_path: '/api/monitoring/metrics'
```

**사용 가능한 메트릭**:
- `sonarqube_project_lines_of_code` (코드 라인 수)
- `sonarqube_project_bugs` (버그 수)
- `sonarqube_project_vulnerabilities` (취약점 수)
- `sonarqube_project_code_smells` (코드 스멜 수)
- `sonarqube_project_coverage` (커버리지)

**Grafana 대시보드**: SonarQube 모니터링을 위해 대시보드 ID `9139`를 가져오세요.

---

## 🛠️ 유지보수

### 데이터 백업

```bash
# 볼륨 백업
docker run --rm \
  -v wealist-sonarqube-data:/data \
  -v $(pwd)/backup:/backup \
  alpine tar czf /backup/sonarqube-data-$(date +%Y%m%d).tar.gz /data
```

### 분석 데이터 삭제

```bash
# Administration → Projects → Management로 이동
# 프로젝트 선택 → Delete
```

### 관리자 비밀번호 재설정

```bash
# SonarQube 중지
make sonar-down

# 데이터베이스를 통한 비밀번호 재설정
docker exec -it wealist-postgres-sonarqube psql -U postgres -d wealist_sonarqube_db -c \
  "UPDATE users SET crypted_password='$2a$12$uCkkXmhW5ThVK8mpBvnXOOJRLd64LJeHTeCkSuB3lfaR2N0AYBaSi', \
   salt=null WHERE login='admin';"
# 이것은 비밀번호를 admin으로 재설정합니다

# SonarQube 재시작
make sonar-up
```

### SonarQube 업데이트

```bash
# 새 이미지 다운로드
docker pull sonarqube:10.4-community

# docker-compose.yml 업데이트
image: sonarqube:10.4-community

# 재시작
make sonar-down
make sonar-up
```

---

## 🚨 문제 해결

### SonarQube가 시작되지 않을 때

**로그 확인**:
```bash
make sonar-logs
```

**일반적인 문제들**:

1. **Elasticsearch bootstrap 검사 실패**
   ```bash
   # docker-compose.yml에서 이미 비활성화됨
   SONAR_ES_BOOTSTRAP_CHECKS_DISABLE=true
   ```

2. **데이터베이스 연결 오류**
   ```bash
   # PostgreSQL 실행 상태 확인
   make sonar-status

   # 데이터베이스 존재 확인
   docker exec -it wealist-postgres-sonarqube psql -U postgres -c "\l" | grep sonarqube
   ```

3. **포트가 이미 사용 중**
   ```bash
   # .env에서 포트 변경
   SONARQUBE_PORT=9001
   ```

### 분석 실패

1. **잘못된 토큰**
   - SonarQube UI에서 토큰 재생성
   - sonar-project.properties 업데이트

2. **네트워크 문제**
   ```bash
   # SonarQube 접근 가능 여부 확인
   curl http://localhost:9000/api/system/status
   ```

3. **커버리지 파일을 찾을 수 없음**
   ```bash
   # 커버리지 파일 존재 확인
   ls -la coverage.out

   # sonar-project.properties에서 경로 확인
   sonar.go.coverage.reportPaths=coverage.out
   ```

---

## 📚 모범 사례

### 1. 정기적인 분석 실행

- **로컬**: 커밋 전
- **CI/CD**: 모든 PR에서
- **스케줄**: 메인 브랜치에서 매일 밤

### 2. 우선순위별 이슈 해결

1. **Blocker**: 애플리케이션을 크래시시키는 버그
2. **Critical**: 보안 취약점
3. **Major**: 심각한 코드 스멜
4. **Minor**: 유지보수성 문제

### 3. 코드 커버리지 목표

- **새 코드**: ≥ 70%
- **전체**: ≥ 60%
- **중요 경로**: ≥ 90%

### 4. 품질 프로필 사용

- **Go**: SonarQube Way (기본값)
- **Java**: SonarQube Way for Java
- **TypeScript**: SonarQube Way for TypeScript

### 5. 보안 핫스팟 처리

- 모든 보안 핫스팟 검토
- 정당한 사유와 함께 "Safe"로 표시하거나 수정
- 검토 없이 무시하지 말 것

---

## 🔗 추가 자료

- **SonarQube 문서**: https://docs.sonarqube.org/latest/
- **Go용 SonarScanner**: https://docs.sonarqube.org/latest/analyzing-source-code/scanners/sonarscanner/
- **품질 게이트**: https://docs.sonarqube.org/latest/user-guide/quality-gates/
- **SonarLint**: https://www.sonarsource.com/products/sonarlint/

---

## 📊 예제: 완전한 워크플로우

### 1. 초기 설정

```bash
# 서비스 시작
make sonar-up

# SonarQube 준비 상태 대기
make sonar-status

# 로그인 및 비밀번호 변경
open http://localhost:9000
```

### 2. 프로젝트 및 토큰 생성

```bash
# UI 또는 API를 통해
curl -X POST -u admin:새로운-비밀번호 \
  "http://localhost:9000/api/projects/create" \
  -d "name=User Service&project=wealist-user-service"

# 토큰 생성
curl -X POST -u admin:새로운-비밀번호 \
  "http://localhost:9000/api/user_tokens/generate" \
  -d "name=user-service-token&projectKey=wealist-user-service"
```

### 3. 코드 분석

```bash
cd services/user-service

# 설정 파일 생성 (토큰을 실제 값으로 교체하세요!)
cat > sonar-project.properties <<EOF
sonar.projectKey=wealist-user-service
sonar.projectName=weAlist User Service
sonar.sources=.
sonar.exclusions=**/*_test.go,**/vendor/**
sonar.go.coverage.reportPaths=coverage.out
sonar.host.url=http://localhost:9000
sonar.token=squ_실제_생성한_토큰_여기에
EOF

# 테스트 및 분석 실행
go test -coverprofile=coverage.out ./...
sonar-scanner
```

**💡 토큰 확인 방법**: SonarQube UI → My Account → Security → Tokens에서 생성한 토큰을 확인할 수 있습니다.

### 4. 결과 검토

```bash
# 프로젝트 열기
open http://localhost:9000/dashboard?id=wealist-user-service
```

---

## 🎯 독립 환경 사용법

### 기본 명령어

```bash
# SonarQube 환경 시작
make sonar-up

# 상태 확인
make sonar-status

# 로그 확인
make sonar-logs

# 환경 중지
make sonar-down

# 환경 재시작
make sonar-restart
```

### 접속 정보

- **SonarQube 웹 UI**: http://localhost:9000
- **PostgreSQL**: localhost:5433 (포트 충돌 방지)
- **기본 로그인**: admin / admin

### 주요 특징

- ✅ **독립 실행**: 기존 전체 환경과 분리
- ✅ **포트 충돌 방지**: PostgreSQL 5433 포트 사용
- ✅ **데이터 지속성**: 컨테이너 재시작 후에도 데이터 유지
- ✅ **자동 헬스체크**: 서비스 준비 상태 자동 확인

---

**상태**: ✅ SonarQube 독립 환경 구축 완료!
**환경**: Docker Compose 독립 실행 (로컬 개발용)
**접속**: http://localhost:9000
