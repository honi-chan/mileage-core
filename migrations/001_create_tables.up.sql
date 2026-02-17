-- users テーブル
CREATE TABLE IF NOT EXISTS users (
    id            CHAR(26)     NOT NULL PRIMARY KEY,
    email         VARCHAR(255) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    created_at    DATETIME(6)  NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at    DATETIME(6)  NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    UNIQUE KEY uk_users_email (email)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- mileage_accounts テーブル（楽観ロック用 version 付き）
CREATE TABLE IF NOT EXISTS mileage_accounts (
    user_id    CHAR(26)    NOT NULL PRIMARY KEY,
    balance    BIGINT      NOT NULL DEFAULT 0,
    version    BIGINT      NOT NULL DEFAULT 0,
    updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    CONSTRAINT fk_mileage_accounts_user FOREIGN KEY (user_id) REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- mileage_transactions テーブル（冪等性キー付き）
CREATE TABLE IF NOT EXISTS mileage_transactions (
    id              CHAR(26)                 NOT NULL PRIMARY KEY,
    user_id         CHAR(26)                 NOT NULL,
    amount          BIGINT                   NOT NULL,
    type            ENUM('grant', 'redeem')  NOT NULL,
    idempotency_key VARCHAR(64)              NOT NULL,
    reason          VARCHAR(255)             NOT NULL DEFAULT '',
    request_id      VARCHAR(64)              NOT NULL DEFAULT '',
    created_at      DATETIME(6)              NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    INDEX idx_transactions_user_created (user_id, created_at),
    UNIQUE KEY uk_transactions_idempotency (user_id, idempotency_key, type),
    CONSTRAINT fk_transactions_user FOREIGN KEY (user_id) REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- achievement_events テーブル
CREATE TABLE IF NOT EXISTS achievement_events (
    id              CHAR(26)     NOT NULL PRIMARY KEY,
    user_id         CHAR(26)     NOT NULL,
    action_type     VARCHAR(64)  NOT NULL,
    idempotency_key VARCHAR(64)  NOT NULL DEFAULT '',
    created_at      DATETIME(6)  NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    INDEX idx_achievements_user (user_id),
    CONSTRAINT fk_achievements_user FOREIGN KEY (user_id) REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
