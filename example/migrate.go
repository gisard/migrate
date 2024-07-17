package main

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/gisard/migrate"

	_ "github.com/go-sql-driver/mysql"
)

// nolint:staticcheck
func main() {
	db, err := sql.Open("mysql", "user:password@tcp(localhost:3306)/controller?multiStatements=true")
	if err != nil {
		panic(err)
	}

	client := migrate.NewMigrate(db, migrate.WithSchemaDir("./example/migrations"))
	client.ApplyObjects(&Migration{db: db, Age: 88})
	err = client.Run(context.WithValue(context.Background(), "age", 18))
	if err != nil {
		fmt.Printf("%+v", err)
	}
}

type Migration struct {
	db  *sql.DB
	Age int64
}

func (m *Migration) Exe1(ctx context.Context) error {
	_, err := m.db.Exec("INSERT INTO user1 (`name`, `age`) VALUES (\"阿东东1\", ?)",
		ctx.Value("age"))
	return err
}

func (m *Migration) Exe2(ctx context.Context) error {
	_, err := m.db.Exec("INSERT INTO user1 (`name`, `age`) VALUES (\"阿东东2\", ?)",
		m.Age)
	return err
}
