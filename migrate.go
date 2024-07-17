package migrate

import (
	"bufio"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"reflect"
	"sync"
)

/*
migrate 是 go 数据迁移工具，可以顺序执行 sql 和 go 方法，对错误进行返回；
需要指定 db 连接和概要表 schemaTable，用于存储已执行的索引和记录错误信息。
*/

var (
	ErrParamIsNotFunc = errors.New("param is not func")
)

const (
	errSchemaRecordIsEmptyFormat   = "schema record %d is empty"
	errSchemaItemIsDuplicateFormat = "schema record %s is duplicate"

	errFuncNameIsDuplicateFormat = "func name %s is duplicate"
	errFileNameIsDuplicateFormat = "file name %s is duplicate"
	errFileIsSameAsFuncFormat    = "file %s is same as func"

	errFileNameNotExistFormat = "file name %s not exist"
	errFuncNameNotExistFormat = "func name %s not exist"

	errFindDirtyIndexFormat           = "find dirty version %d"
	errSchemaVersionLargeRecordFormat = "schema version %d large than current record"

	errFileTypeNotSupportedFormat = "file type %s not supported"

	errFuncFormatNotCorrectFormat = "func format %s not correct, should be func(ctx context.Context) error"
)

const (
	insertDefaultSchema = "INSERT INTO %s (`version`, `dirty`) VALUES (0, 0)"
)

const (
	createSchemaTableFormat = "CREATE TABLE IF NOT EXISTS %s (`version` int NOT NULL DEFAULT 0, `dirty` tinyint(1) NOT NULL DEFAULT 0) ENGINE=InnoDB;"

	querySchemaRecordFormat = "SELECT `version`, `dirty` FROM %s LIMIT 1"

	updateSchemaQuery = "UPDATE %s SET `version` = ?"

	updateDirtyQuery = "UPDATE %s SET `version` = ?, `dirty` = ?"
)

const (
	sqlFileExt = ".sql"
	funcExt    = ""
)

const (
	fileJoinFormat = "%s/%s"
)

const (
	defaultSchemaTableName = "schema_migrations"
	defaultSchemaDir       = "./migrations"
	defaultSchemaFile      = "migrate.txt"
)

type Migrate interface {
	AddHandlers(handlers ...Handler)

	Run(ctx context.Context) error
	ApplyObjects(objects ...interface{})
}

func NewMigrate(db *sql.DB, options ...Option) Migrate {
	m := &migrate{
		db:          db,
		schemaDir:   defaultSchemaDir,
		schemaFile:  defaultSchemaFile,
		schemaTable: defaultSchemaTableName,
	}
	for _, option := range options {
		option(m)
	}
	return m
}

type migrate struct {
	mutex sync.Mutex

	db *sql.DB // db 连接

	schemaDir   string // 概要目录
	schemaFile  string // 概要文件，用于记录处理单元顺序，默认存在于概要目录下
	schemaTable string // 概要表

	applyObjects []interface{}

	handlers []Handler // 运行单元列表
}

func (m *migrate) AddHandlers(handlers ...Handler) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.handlers = append(m.handlers, handlers...)
}

func (m *migrate) ApplyObjects(objects ...interface{}) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.applyObjects = append(m.applyObjects, objects...)
}

func (m *migrate) Run(ctx context.Context) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	// 1.读取概要文件，中间不能有空行，校验执行名称非重复
	items, err := m.getSchemaItems()
	if err != nil {
		return err
	}
	// 2.获取需要执行函数的结构体，校验函数名称非重复
	methodMap, err := m.getMethods()
	if err != nil {
		return err
	}
	// 3.获取需要执行的文件，校验文件名称非重复
	fileNameMap, err := m.getFileNames()
	if err != nil {
		return err
	}
	for fileName := range fileNameMap {
		if _, ok := methodMap[fileName]; ok {
			return fmt.Errorf(fmt.Sprintf(errFileIsSameAsFuncFormat, fileName))
		}
	}
	// 4.读取概要表，获取执行记录
	record, err := m.getSchemaRecord()
	if err != nil {
		return err
	}
	if record.dirty {
		return fmt.Errorf(fmt.Sprintf(errFindDirtyIndexFormat, record.version))
	}
	if len(items) < record.version {
		return fmt.Errorf(fmt.Sprintf(errSchemaVersionLargeRecordFormat, record.version))
	}
	// 5.获取执行记录位置，从该位置开始构建 handlers
	for i := record.version; i < len(items); i++ {
		item := items[i]

		ext := path.Ext(item)
		switch ext {
		case sqlFileExt:
			if _, ok := fileNameMap[item]; !ok {
				return fmt.Errorf(errFileNameNotExistFormat, item)
			}
			sqlContent, err := getFileContent(fmt.Sprintf(fileJoinFormat, m.schemaDir, item))
			if err != nil {
				return err
			}
			m.handlers = append(m.handlers, newSQLHandler(m.db, i+1, sqlContent))
		case funcExt:
			if _, ok := methodMap[item]; !ok {
				return fmt.Errorf(errFuncNameNotExistFormat, item)
			}
			method := methodMap[item]
			// 校验方法的参数，格式必须满足 func(context.Context) error
			err = checkMethodFormat(&method)
			if err != nil {
				return err
			}
			m.handlers = append(m.handlers, newGoHandler(i+1, method.value))
		default:
			// 后续可自行扩展其他类型
			return fmt.Errorf(errFileTypeNotSupportedFormat, item)
		}
	}
	// 6.依次执行 handlers
	return m.run(ctx)
}

func (m *migrate) getSchemaItems() ([]string, error) {
	schemaFileName := fmt.Sprintf(fileJoinFormat, m.schemaDir, m.schemaFile)
	schemaFile, err := os.Open(schemaFileName)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			err = os.MkdirAll(path.Dir(schemaFileName), os.ModePerm)
			if err != nil {
				return nil, err
			}
			schemaFile, err = os.Create(schemaFileName)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	scanner := bufio.NewScanner(schemaFile)
	itemMap := make(map[string]struct{})
	var (
		index       int
		schemaItems []string
	)

	for scanner.Scan() {
		index++
		line := scanner.Text()
		if line == "" {
			if !scanner.Scan() {
				break
			}
			return nil, fmt.Errorf(errSchemaRecordIsEmptyFormat, index)
		}
		if _, ok := itemMap[line]; ok {
			return nil, fmt.Errorf(errSchemaItemIsDuplicateFormat, line)
		}
		itemMap[line] = struct{}{}
		schemaItems = append(schemaItems, line)
	}
	return schemaItems, nil
}

func (m *migrate) getSchemaRecord() (*schema, error) {
	err := m.initSchemaTable()
	if err != nil {
		return nil, err
	}

	var sche schema
	err = m.db.QueryRow(fmt.Sprintf(querySchemaRecordFormat, m.schemaTable)).
		Scan(&sche.version, &sche.dirty)
	if err == nil {
		return &sche, nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		_, err = m.db.Exec(fmt.Sprintf(insertDefaultSchema, m.schemaTable))
		if err != nil {
			return nil, err
		}
	}
	return &sche, err
}

func (m *migrate) initSchemaTable() error {
	_, err := m.db.Exec(fmt.Sprintf(createSchemaTableFormat, m.schemaTable))
	return err
}

func (m *migrate) getMethods() (map[string]method, error) {
	methods := make(map[string]method)
	for _, object := range m.applyObjects {
		objType := reflect.TypeOf(object)
		objValue := reflect.ValueOf(object)
		for i := 0; i < objType.NumMethod(); i++ {
			mt := objType.Method(i)
			if _, ok := methods[mt.Name]; ok {
				return nil, fmt.Errorf(errFuncNameIsDuplicateFormat, mt.Name)
			}
			methods[mt.Name] = method{
				name:  mt.Name,
				value: objValue.MethodByName(mt.Name),
			}
		}
	}
	return methods, nil
}

func (m *migrate) getFileNames() (map[string]struct{}, error) {
	entries, err := os.ReadDir(m.schemaDir)
	if err != nil {
		return nil, err
	}

	fileNameMap := make(map[string]struct{})
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if _, ok := fileNameMap[entry.Name()]; ok {
			return nil, fmt.Errorf(errFileNameIsDuplicateFormat, entry.Name())
		}
		fileNameMap[entry.Name()] = struct{}{}
	}
	return fileNameMap, nil
}

func (m *migrate) run(ctx context.Context) error {
	for _, handler := range m.handlers {
		err := handler.Exec(ctx)
		if err != nil {
			// 发生错误时，记录 dirty 到 schema 表
			_, innerErr := m.db.Exec(fmt.Sprintf(updateDirtyQuery, m.schemaTable),
				handler.GetIndex(), 1)
			if innerErr != nil {
				return innerErr
			}
			return err
		}
		// 成功时更新 version 字段
		_, err = m.db.Exec(fmt.Sprintf(updateSchemaQuery, m.schemaTable),
			handler.GetIndex())
		if err != nil {
			return err
		}
	}
	return nil
}

func getFileContent(fileName string) (string, error) {
	file, err := os.Open(fileName)
	if err != nil {
		return "", err
	}
	defer file.Close()
	content, err := io.ReadAll(file)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

func checkMethodFormat(mh *method) error {
	if mh.value.Kind() != reflect.Func {
		return ErrParamIsNotFunc
	}

	if mh.value.Type().NumIn() != 1 || mh.value.Type().NumOut() != 1 {
		return fmt.Errorf(errFuncFormatNotCorrectFormat, mh.name)
	}
	paramInType := mh.value.Type().In(0)
	paramOutType := mh.value.Type().Out(0)
	if paramInType != reflect.TypeOf((*context.Context)(nil)).Elem() ||
		paramOutType != reflect.TypeOf((*error)(nil)).Elem() {
		return fmt.Errorf(errFuncFormatNotCorrectFormat, mh.name)
	}
	return nil
}

type method struct {
	name  string
	value reflect.Value
}

type schema struct {
	version int
	dirty   bool
}

type Option func(m *migrate)

func WithSchemaTable(tableName string) Option {
	return func(m *migrate) {
		m.schemaTable = tableName
	}
}

func WithSchemaDir(dir string) Option {
	return func(m *migrate) {
		m.schemaDir = dir
	}
}

func WithSchemaFile(file string) Option {
	return func(m *migrate) {
		m.schemaFile = file
	}
}
