package db

import (
	"context"
	"database/sql"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

type Client struct {
	*sqlx.DB
	queryTracer pgx.QueryTracer
}

func Open(ctx context.Context, dsn string, opts ...Option) (*Client, error) {
	db := &Client{}
	for _, opt := range opts {
		opt(db)
	}

	cfg, err := pgx.ParseConfig(dsn)
	if err != nil {
		return nil, errors.WithMessage(err, "parse config")
	}

	cfg.Tracer = db.queryTracer
	sqlDb := stdlib.OpenDB(*cfg)

	pgDb := sqlx.NewDb(sqlDb, "pgx")
	if err != nil {
		return nil, errors.WithMessage(err, "open database with pgx driver")
	}
	pgDb.MapperFunc(ToSnakeCase)
	err = pgDb.PingContext(ctx)
	if err != nil {
		return nil, errors.WithMessage(err, "ping database")
	}

	db.DB = pgDb
	return db, nil
}

func (db *Client) RunInTransaction(ctx context.Context, txFunc TxFunc, opts ...TxOption) (err error) {
	options := &txOptions{}
	for _, opt := range opts {
		opt(options)
	}
	tx, err := db.BeginTxx(ctx, options.nativeOpts)
	if err != nil {
		return errors.WithMessage(err, "begin transaction")
	}
	defer func() {
		p := recover()
		if p != nil { // do rollback and repanic
			_ = tx.Rollback()
			panic(p)
		}

		if err != nil {
			rbErr := tx.Rollback()
			if rbErr != nil {
				err = errors.WithMessage(err, rbErr.Error())
			}
			return
		}

		err = tx.Commit()
		if err != nil {
			err = errors.WithMessage(err, "commit tx")
		}
	}()

	return txFunc(ctx, &Tx{tx})
}

func (db *Client) Select(ctx context.Context, ptr interface{}, query string, args ...interface{}) error {
	return db.SelectContext(ctx, ptr, query, args...)
}

func (db *Client) SelectRow(ctx context.Context, ptr interface{}, query string, args ...interface{}) error {
	return db.GetContext(ctx, ptr, query, args...)
}

func (db *Client) Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return db.ExecContext(ctx, query, args...)
}

func (db *Client) ExecNamed(ctx context.Context, query string, arg interface{}) (sql.Result, error) {
	return db.NamedExecContext(ctx, query, arg)
}
