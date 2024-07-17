package migrate

import (
	"context"
	"database/sql"
	"fmt"
)

const (
	ErrQueryWithIndexFormat = "%s; query err with index is %d"
)

func newSQLHandler(db *sql.DB, index int, query string) Handler {
	return &sqlHandler{
		index: index,
		query: query,
		db:    db,
	}
}

// sqlHandler 包含具体 sql 语句
type sqlHandler struct {
	index int
	query string
	db    *sql.DB
}

func (s *sqlHandler) GetIndex() int {
	return s.index
}

func (s *sqlHandler) Exec(ctx context.Context) error {
	if s.query == "" {
		return nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	_, err = tx.Exec(s.query)
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf(ErrQueryWithIndexFormat, err.Error(), s.index)
	}
	_ = tx.Commit()
	return nil
}
