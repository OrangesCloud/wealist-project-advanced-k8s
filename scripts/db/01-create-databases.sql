-- =============================================================================
-- wealist Database & User Creation Script
-- =============================================================================
-- Run as postgres superuser:
--   sudo -u postgres psql -f 01-create-databases.sql
-- =============================================================================

-- -----------------------------------------------------------------------------
-- Create Users (Roles)
-- -----------------------------------------------------------------------------
DO $$
BEGIN
    -- user-service
    IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'user_service') THEN
        CREATE ROLE user_service WITH LOGIN PASSWORD 'user_service_password';
    END IF;

    -- board-service
    IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'board_service') THEN
        CREATE ROLE board_service WITH LOGIN PASSWORD 'board_service_password';
    END IF;

    -- chat-service
    IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'chat_service') THEN
        CREATE ROLE chat_service WITH LOGIN PASSWORD 'chat_service_password';
    END IF;

    -- noti-service
    IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'noti_service') THEN
        CREATE ROLE noti_service WITH LOGIN PASSWORD 'noti_service_password';
    END IF;

    -- storage-service
    IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'storage_service') THEN
        CREATE ROLE storage_service WITH LOGIN PASSWORD 'storage_service_password';
    END IF;
END
$$;

-- -----------------------------------------------------------------------------
-- Create Databases
-- -----------------------------------------------------------------------------
-- Note: CREATE DATABASE cannot be inside a transaction block, so we use \gexec

SELECT 'CREATE DATABASE user_db OWNER user_service'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'user_db')\gexec

SELECT 'CREATE DATABASE board_db OWNER board_service'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'board_db')\gexec

SELECT 'CREATE DATABASE chat_db OWNER chat_service'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'chat_db')\gexec

SELECT 'CREATE DATABASE noti_db OWNER noti_service'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'noti_db')\gexec

SELECT 'CREATE DATABASE storage_db OWNER storage_service'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'storage_db')\gexec

-- -----------------------------------------------------------------------------
-- Grant Privileges
-- -----------------------------------------------------------------------------
GRANT ALL PRIVILEGES ON DATABASE user_db TO user_service;
GRANT ALL PRIVILEGES ON DATABASE board_db TO board_service;
GRANT ALL PRIVILEGES ON DATABASE chat_db TO chat_service;
GRANT ALL PRIVILEGES ON DATABASE noti_db TO noti_service;
GRANT ALL PRIVILEGES ON DATABASE storage_db TO storage_service;

-- -----------------------------------------------------------------------------
-- Enable UUID Extension (required for all databases)
-- -----------------------------------------------------------------------------
\c user_db
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
GRANT ALL ON SCHEMA public TO user_service;

\c board_db
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
GRANT ALL ON SCHEMA public TO board_service;

\c chat_db
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
GRANT ALL ON SCHEMA public TO chat_service;

\c noti_db
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
GRANT ALL ON SCHEMA public TO noti_service;

\c storage_db
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
GRANT ALL ON SCHEMA public TO storage_service;

\echo '✅ All databases and users created successfully!'
