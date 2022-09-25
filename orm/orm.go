package orm

import (
	"database/sql"
	"errors"
	"fmt"
	grLog "github.com/Jack-ZL/go_rookie/log"
	"reflect"
	"strings"
	"time"
)

type GrDb struct {
	db     *sql.DB
	logger *grLog.Logger
	Prefix string // 表名前缀
}

type GrSession struct {
	db          *GrDb
	tableName   string
	fieldName   []string
	placeHolder []string
	values      []any
}

/**
 * Open
 * @Author：Jack-Z
 * @Description: 连接数据库
 * @param driverName
 * @param source
 */
func Open(driverName string, source string) *GrDb {
	db, err := sql.Open(driverName, source)
	if err != nil {
		panic(err)
	}

	db.SetMaxIdleConns(5)                  // 最大空闲连接数，默认不配置，是2个最大空闲连接
	db.SetMaxOpenConns(100)                // 最大连接数，默认不配置，是不限制最大连接数
	db.SetConnMaxLifetime(time.Minute * 3) // 连接最大存活时间
	db.SetConnMaxIdleTime(time.Minute * 1) // 空闲连接最大存活时间

	grdb := &GrDb{
		db:     db,
		logger: grLog.Default(),
	}
	err = db.Ping()
	if err != nil {
		panic(err)
	}
	return grdb
}

/**
 * SetMaxIdleConns
 * @Author：Jack-Z
 * @Description: 最大空闲连接数，默认不配置，是2个最大空闲连接
 * @receiver db
 * @param max
 */
func (db *GrDb) SetMaxIdleConns(max int) {
	db.db.SetMaxIdleConns(max)
}

/**
 * Close
 * @Author：Jack-Z
 * @Description:
 * @receiver db*
 * @return error
 */
func (db *GrDb) Close() error {
	return db.db.Close()
}

/**
 * New
 * @Author：Jack-Z
 * @Description: new一个db连接
 * @receiver db
 * @return *GrSession
 */
func (db *GrDb) New() *GrSession {
	return &GrSession{
		db: db,
	}
}

/**
 * Table
 * @Author：Jack-Z
 * @Description: 指定表名
 * @receiver s
 * @param tableName
 * @return *GrSession
 */
func (s *GrSession) Table(tableName string) *GrSession {
	s.tableName = tableName
	return s
}

/**
 * Insert
 * @Author：Jack-Z
 * @Description: （单条）插入数据操作
 * @receiver s
 * @param data
 * @return int64 插入数据的id
 * @return int64 受影响的行数
 * @return error 错误信息
 */
func (s *GrSession) Insert(data any) (int64, int64, error) {
	// 每一个操作都独立，互不影响，即在一个会话内完成
	s.fieldNames(data)
	query := fmt.Sprintf("insert into %s (%s) values (%s)",
		s.tableName,
		strings.Join(s.fieldName, ","),
		strings.Join(s.placeHolder, ","),
	)
	s.db.logger.Info(query)
	sp, err := s.db.db.Prepare(query)
	if err != nil {
		return -1, -1, err
	}
	res, err := sp.Exec(s.values...)
	if err != nil {
		return -1, -1, err
	}
	last_id, err := res.LastInsertId() // 获取插入的主键id
	if err != nil {
		return -1, -1, err
	}

	affected, err := res.RowsAffected() // 受影响行数
	if err != nil {
		return -1, -1, err
	}
	return last_id, affected, err
}

/**
 * fieldNames
 * @Author：Jack-Z
 * @Description: insert的字段处理
 * @receiver s
 * @param data
 */
func (s *GrSession) fieldNames(data any) {
	t := reflect.TypeOf(data)
	v := reflect.ValueOf(data)
	if t.Kind() != reflect.Pointer {
		panic(errors.New("data must be pointer"))
	}

	tVar := t.Elem()
	vVar := v.Elem()
	if s.tableName == "" {
		s.tableName = s.db.Prefix + strings.ToLower(Name(tVar.Name()))
	}
	for i := 0; i < tVar.NumField(); i++ {
		fieldName := tVar.Field(i).Name
		tag := tVar.Field(i).Tag
		sqlTag := tag.Get("grorm")
		if sqlTag == "" {
			sqlTag = strings.ToLower(Name(fieldName))
		} else {
			if strings.Contains(sqlTag, "auto_increment") {
				// 自增长的主键id
				continue
			}

			if strings.Contains(sqlTag, ",") {
				sqlTag = sqlTag[:strings.Index(sqlTag, ",")]
			}
		}
		id := vVar.Field(i).Interface()
		// 对id做个判断，如果其值小于等于0，数据库可能是自增，跳过此字段
		if strings.ToLower(sqlTag) == "id" && IsAutoId(id) {
			continue
		}

		s.fieldName = append(s.fieldName, sqlTag)
		s.placeHolder = append(s.placeHolder, "?")
		s.values = append(s.values, vVar.Field(i).Interface())
	}
}

/**
 * IsAutoId
 * @Author：Jack-Z
 * @Description: 是否为主键id
 * @param id
 * @return bool
 */
func IsAutoId(id any) bool {
	t := reflect.TypeOf(id)
	switch t.Kind() {
	case reflect.Int64:
		if id.(int64) <= 0 {
			return true
		}

	case reflect.Int32:
		if id.(int32) <= 0 {
			return true
		}

	case reflect.Int:
		if id.(int) <= 0 {
			return true
		}

	default:
		return false
	}
	return false
}

/**
 * Name
 * @Author：Jack-Z
 * @Description: 通过反射将“UserName”驼峰格式转为“User_Name”
 * @param name
 * @return string
 */
func Name(name string) string {
	names := name[:] // 字符串分割为单个字母的切片
	lastIndex := 0
	var sb strings.Builder
	for index, value := range names {
		if value >= 65 && value <= 90 { // 如果是大写字母
			if index == 0 {
				continue
			}
			sb.WriteString(name[:index])
			sb.WriteString("_")
			lastIndex = index
		}
	}
	sb.WriteString(name[lastIndex:])
	return sb.String()
}
