package MiddleWare

import (
	"context"
	"database/sql"
	"net/http"
)

type dbKeyType string

const dbKey dbKeyType = "dbConn"

func AttachDB(ctx context.Context, db *sql.DB) context.Context {
	return context.WithValue(ctx, dbKey, db)
}

func GetDB(ctx context.Context) (*sql.DB, bool) {
	db, ok := ctx.Value(dbKey).(*sql.DB)
	return db, ok
}

func DBMiddleware(ctx context.Context, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r = r.WithContext(context.WithValue(r.Context(), dbKey, ctx.Value(dbKey)))
		next(w, r)
	}
}
