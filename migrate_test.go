package migrate

import (
	"context"
	"fmt"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewMigrateSchema(t *testing.T) {
	// 测试 schema 重复
	db, _, err := sqlmock.New()
	if err != nil {
		panic(err)
	}
	defer db.Close()

	// 测试 schema 重复
	dumpClient := NewMigrate(db, WithSchemaDir("./test/migrationsdump"))
	err = dumpClient.Run(context.Background())
	assert.Equal(t, fmt.Errorf(errSchemaItemIsDuplicateFormat, "1.sql"), err)
	// 测试 schema 有空行
	emptyLineClient := NewMigrate(db, WithSchemaDir("./test/migrationsempty"))
	err = emptyLineClient.Run(context.Background())
	assert.Equal(t, fmt.Errorf(errSchemaRecordIsEmptyFormat, 2), err)
}

type Object1 struct {
}

func (o *Object1) Run(ctx context.Context) error {
	return nil
}

type Object2 struct {
}

func (o *Object2) Run(ctx context.Context) error {
	return nil
}

func TestNewMigrateFunc(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		panic(err)
	}
	defer db.Close()

	normalClient := NewMigrate(db, WithSchemaDir("./test/migrationsnormal"))
	normalClient.ApplyObjects(&Object1{}, &Object2{})

	err = normalClient.Run(context.Background())
	assert.Equal(t, fmt.Errorf(errFuncNameIsDuplicateFormat, "Run"), err)
}

func TestNewMigrateSchemaFiles(t *testing.T) {
	// 没办法 mock create table 语句，后续先不能测试了
	//db, mock, err := sqlmock.New()
	//if err != nil {
	//	// 错误处理
	//}
	//defer db.Close()
	//
	//mock.ExpectExec("CREATE TABLE schema_migrations").WillReturnError(err)
	//
	//mock.ExpectQuery("SELECT `version`, `dirty` FROM schema_migrations LIMIT 1").
	//	WillReturnRows(sqlmock.NewRows([]string{"version", "dirty"}).AddRow(0, 0))
	//// 测试 schema 类型错误
	//invalidTypeClient := NewMigrate(db, WithSchemaDir("./test/migrationsinvalidtype"))
	//err = invalidTypeClient.Run(context.Background())
	//assert.Equal(t, fmt.Errorf(errFileTypeNotSupportedFormat, "2.mongodb"), err)
}
