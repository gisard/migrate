# migrate

migrate 是一个 Go 语言编写的数据迁移工具，它允许按照顺序执行 SQL 语句和 Go 方法，并能够处理执行过程中的错误。该工具会记录已执行的迁移索引，并在迁移过程中出现错误时提供详细信息，方便开发者进行故障排查和恢复。

## 特性

- 支持顺序执行 SQL 脚本和 Go 方法
- 通过数据库表记录迁移历史，避免重复执行
- 支持迁移过程中的错误处理和状态管理
- 可以轻松扩展支持更多类型的处理程序
- 提供灵活的配置选项

## 安装

```bash
go get github.com/gisard/migrate
```

## 基本用法

```go
package main

import (
    "context"
    "database/sql"
    "fmt"
    "github.com/gisard/migrate"
    
    _ "github.com/go-sql-driver/mysql"
)

func main() {
    // 建立数据库连接
    db, err := sql.Open("mysql", "user:password@tcp(localhost:3306)/dbname?multiStatements=true")
    if err != nil {
        panic(err)
    }
    
    // 创建迁移客户端
    client := migrate.NewMigrate(db, 
        migrate.WithSchemaDir("./migrations"), // 可选，指定迁移文件目录
        migrate.WithSchemaTable("schema_migrations"), // 可选，指定存储迁移记录的表名
    )
    
    // 注册包含迁移方法的结构体
    client.ApplyObjects(&Migration{db: db})
    
    // 执行迁移
    err = client.Run(context.Background())
    if err != nil {
        fmt.Printf("迁移失败: %v\n", err)
    }
}

// 迁移结构体示例
type Migration struct {
    db *sql.DB
}

// 迁移方法示例
func (m *Migration) CreateUsers(ctx context.Context) error {
    _, err := m.db.Exec("INSERT INTO users (name) VALUES ('admin')")
    return err
}
```

## 详细指南

### 1. 迁移列表

- 在迁移目录中需要有一个迁移列表文件（默认为 `migrate.txt`），用于存储执行列表。
- 执行列表按照顺序包含所有需要执行的项，每行一个，不允许有空行。
- 以 `.sql` 结尾的项目表示 SQL 语句文件。
- 无后缀的项表示 Go 方法名。

示例 `migrate.txt`:
```
1_create_tables.sql
CreateAdmin
2_add_indexes.sql
SeedInitialData
```

### 2. SQL 文件目录

- 可以自由指定 SQL 文件的路径，默认为 `./migrations`。
- SQL 文件会按照迁移列表中指定的顺序执行。

### 3. Go 方法

- 迁移客户端可以接受结构体或指针，它会从所有注册的结构体或指针中搜索迁移方法。
- 通过反射调用与名称匹配的方法，并传入上下文参数。
- 方法格式必须为 `func(ctx context.Context) error`。

### 4. 扩展

- 通过实现 `Handler` 接口可以扩展支持其他类型的处理程序。
- 不同类型的处理程序应当通过后缀加以区分。
- 在构造所有类型的处理程序时添加相应代码。

## 配置选项

migrate 提供了几个配置选项，可以在创建迁移客户端时使用：

- `WithSchemaTable(tableName string)`: 设置存储迁移记录的表名，默认为 `schema_migrations`。
- `WithSchemaDir(dir string)`: 设置迁移文件的目录，默认为 `./migrations`。
- `WithSchemaFile(file string)`: 设置迁移列表文件的名称，默认为 `migrate.txt`。

## 完整示例

请参考项目中的 `example` 目录，其中包含了一个完整的迁移示例。