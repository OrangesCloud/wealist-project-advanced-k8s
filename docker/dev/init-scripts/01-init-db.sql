-- =============================================================================
-- Wealist Dev Database Initialization
-- =============================================================================
-- 이 스크립트는 PostgreSQL 컨테이너 최초 시작 시 자동 실행됩니다.

-- 확장 기능 활성화
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";

-- 기본 스키마 설정
SET timezone TO 'Asia/Seoul';

-- 애플리케이션에서 사용할 추가 사용자 (선택사항)
-- DO $$
-- BEGIN
--     IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'wealist_app') THEN
--         CREATE ROLE wealist_app WITH LOGIN PASSWORD 'app-password';
--         GRANT ALL PRIVILEGES ON DATABASE wealist TO wealist_app;
--     END IF;
-- END $$;

-- 초기화 완료 로그
DO $$
BEGIN
    RAISE NOTICE 'Wealist Dev Database initialized successfully!';
END $$;
