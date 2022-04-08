package repository

import (
	"context"
	"github.com/diez37/go-packages/clients/db"
	"github.com/doug-martin/goqu/v9"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"time"
)

const (
	sqlTableName = "refresh_tokens"
)

type sql struct {
	db     goqu.SQLDatabase
	tracer trace.Tracer
}

func NewSql(db goqu.SQLDatabase, tracer trace.Tracer) Repository {
	return &sql{db: db, tracer: tracer}
}

func (repository *sql) FindByLogin(ctx context.Context, login uuid.UUID) ([]*RefreshToken, error) {
	ctx, span := repository.tracer.Start(ctx, "finder.login")
	defer span.End()

	span.SetAttributes(
		attribute.String("login", login.String()),
		attribute.String("repository", "sql"),
	)

	sql, args, err := goqu.From(sqlTableName).
		Where(goqu.I("login").Eq(login.String())).
		Order(goqu.I("created_at").Asc()).
		ToSQL()
	if err != nil {
		return nil, err
	}

	rows, err := repository.db.QueryContext(ctx, sql, args...)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var refreshTokens []*RefreshToken

	for rows.Next() {
		refreshToken := &RefreshToken{}

		if err := rows.Scan(&refreshToken.UUID, &refreshToken.Login, &refreshToken.Ip, &refreshToken.Fingerprint, &refreshToken.UserAgent, &refreshToken.CreatedAt, &refreshToken.ExpiresIn); err != nil {
			return nil, err
		}

		refreshTokens = append(refreshTokens, refreshToken)
	}

	if len(refreshTokens) == 0 {
		return nil, db.RecordNotFoundError
	}

	return refreshTokens, nil
}

func (repository *sql) FindByUUID(ctx context.Context, uuid uuid.UUID) (*RefreshToken, error) {
	ctx, span := repository.tracer.Start(ctx, "finder.uuid")
	defer span.End()

	span.SetAttributes(
		attribute.String("uuid", uuid.String()),
		attribute.String("repository", "sql"),
	)

	sql, args, err := goqu.From(sqlTableName).Where(goqu.I("uuid").Eq(uuid.String())).ToSQL()
	if err != nil {
		return nil, err
	}

	rows, err := repository.db.QueryContext(ctx, sql, args...)
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		refreshToken := &RefreshToken{}

		if err := rows.Scan(&refreshToken.UUID, &refreshToken.Login, &refreshToken.Ip, &refreshToken.Fingerprint, &refreshToken.UserAgent, &refreshToken.CreatedAt, &refreshToken.ExpiresIn); err != nil {
			return nil, err
		}

		return refreshToken, nil
	}

	return nil, db.RecordNotFoundError
}

func (repository *sql) Insert(ctx context.Context, tokens ...*RefreshToken) error {
	ctx, span := repository.tracer.Start(ctx, "saver.insert")
	defer span.End()

	span.SetAttributes(
		attribute.Int("length", len(tokens)),
		attribute.String("repository", "sql"),
	)

	rows := make([]interface{}, len(tokens))

	now := time.Now().In(time.UTC)

	for index, token := range tokens {
		token.CreatedAt = now
		rows[index] = token
	}

	sql, args, err := goqu.Insert(sqlTableName).Rows(rows...).ToSQL()
	if err != nil {
		return err
	}

	_, err = repository.db.ExecContext(ctx, sql, args...)

	return err
}

func (repository *sql) BlockByUUID(ctx context.Context, uuids ...uuid.UUID) error {
	if len(uuids) == 0 {
		return nil
	}

	ctx, span := repository.tracer.Start(ctx, "blocker.uuid")
	defer span.End()

	span.SetAttributes(
		attribute.Int("length", len(uuids)),
		attribute.String("repository", "sql"),
	)

	sql, args, err := goqu.Delete(sqlTableName).Where(goqu.Ex{"uuid": uuids}).ToSQL()
	if err != nil {
		return err
	}

	_, err = repository.db.ExecContext(ctx, sql, args...)

	return err
}

func (repository *sql) BlockByDate(ctx context.Context, date time.Time) error {
	ctx, span := repository.tracer.Start(ctx, "blocker.date")
	defer span.End()

	span.SetAttributes(attribute.String("repository", "sql"))

	sql, args, err := goqu.Delete(sqlTableName).Where(goqu.I("expires_in").Lte(date)).ToSQL()
	if err != nil {
		return err
	}

	_, err = repository.db.ExecContext(ctx, sql, args...)

	return err
}
