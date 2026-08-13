package api

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// Connect opens a MySQL pool from BLUEPRINT_DB_* env vars and waits (with
// retries) until the server is reachable — the DB container may still be
// starting when the app boots.
func Connect() (*sql.DB, error) {
	dsn := fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?parseTime=true&charset=utf8mb4&loc=Local",
		os.Getenv("BLUEPRINT_DB_USERNAME"),
		os.Getenv("BLUEPRINT_DB_PASSWORD"),
		envOr("BLUEPRINT_DB_HOST", "localhost"),
		envOr("BLUEPRINT_DB_PORT", "3306"),
		os.Getenv("BLUEPRINT_DB_DATABASE"),
	)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	db.SetConnMaxLifetime(time.Hour)
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)

	for i := 0; i < 15; i++ {
		if err = db.Ping(); err == nil {
			return db, nil
		}
		time.Sleep(2 * time.Second)
	}
	return nil, fmt.Errorf("database not reachable: %w", err)
}

// Migrate creates the schema if it does not exist (idempotent, runs on boot).
func Migrate(db *sql.DB) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id            CHAR(36)     NOT NULL PRIMARY KEY,
			name          VARCHAR(120) NOT NULL,
			email         VARCHAR(190) NOT NULL UNIQUE,
			password_hash VARCHAR(255) NOT NULL,
			created_at    TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,

		`CREATE TABLE IF NOT EXISTS categories (
			id         CHAR(36)                      NOT NULL PRIMARY KEY,
			user_id    CHAR(36)                      NOT NULL,
			name       VARCHAR(120)                  NOT NULL,
			type       ENUM('income','expense')      NOT NULL,
			color      VARCHAR(16)                   NOT NULL DEFAULT '#64748b',
			icon       VARCHAR(40)                   NOT NULL DEFAULT 'Tag',
			created_at TIMESTAMP                     NOT NULL DEFAULT CURRENT_TIMESTAMP,
			INDEX idx_categories_user (user_id),
			CONSTRAINT fk_categories_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,

		`CREATE TABLE IF NOT EXISTS transactions (
			id          CHAR(36)                 NOT NULL PRIMARY KEY,
			user_id     CHAR(36)                 NOT NULL,
			type        ENUM('income','expense') NOT NULL,
			amount      BIGINT                   NOT NULL,
			category_id CHAR(36)                 NULL,
			note        VARCHAR(255)             NOT NULL DEFAULT '',
			date        DATE                     NOT NULL,
			created_at  TIMESTAMP                NOT NULL DEFAULT CURRENT_TIMESTAMP,
			INDEX idx_tx_user_date (user_id, date),
			CONSTRAINT fk_tx_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
			CONSTRAINT fk_tx_category FOREIGN KEY (category_id) REFERENCES categories(id) ON DELETE SET NULL
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,

		`CREATE TABLE IF NOT EXISTS budgets (
			id          CHAR(36) NOT NULL PRIMARY KEY,
			user_id     CHAR(36) NOT NULL,
			category_id CHAR(36) NOT NULL,
			limit_amount BIGINT  NOT NULL,
			month       CHAR(7)  NOT NULL,
			created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE KEY uq_budget (user_id, category_id, month),
			CONSTRAINT fk_budget_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
			CONSTRAINT fk_budget_category FOREIGN KEY (category_id) REFERENCES categories(id) ON DELETE CASCADE
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			return fmt.Errorf("migrate: %w", err)
		}
	}
	return nil
}
