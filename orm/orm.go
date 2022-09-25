package orm

import (
	"database/sql"
	"time"
)

type GrDb struct {
	db           *sql.DB
	MaxIdleConns int
}

type GrSession struct {
	db        *GrDb
	tableName string
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
		db: db,
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
 * @Description: 插入数据操作
 * @receiver db
 * @param data
 */
func (db *GrSession) Insert(data any) {
	// 每一个操作都独立，互不影响，即在一个会话内完成
}
