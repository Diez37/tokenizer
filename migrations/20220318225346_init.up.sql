CREATE TABLE IF NOT EXISTS refresh_tokens
(
    uuid        CHAR(36)     NOT NULL PRIMARY KEY,
    login       CHAR(36)     NOT NULL,
    ip          VARCHAR(45)  NOT NULL,
    fingerprint VARCHAR(256) NOT NULL,
    user_agent  VARCHAR(256) NOT NULL,
    created_at  TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_in  TIMESTAMP    NOT NULL
    );

CREATE UNIQUE INDEX refresh_tokens_login_uuid ON refresh_tokens (login, uuid);
