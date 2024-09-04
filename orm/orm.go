package orm

import (
	"database/sql"
	"errors"
	"fmt"
	grLog "github.com/Jack-ZL/go_rookie/log"
	_ "github.com/go-sql-driver/mysql"
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
	tx          *sql.Tx
	beginTx     bool
	tableName   string
	fieldName   []string
	placeHolder []string
	values      []any
	whereValues []any
	updateParam strings.Builder
	whereParam  strings.Builder
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
 * Close 关闭连接
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
func (db *GrDb) New(data any) *GrSession {
	s := &GrSession{
		db: db,
	}

	t := reflect.TypeOf(data)
	if t.Kind() != reflect.Pointer {
		panic(errors.New("data must be pointer"))
	}

	tVar := t.Elem()
	if s.tableName == "" {
		s.tableName = s.db.Prefix + strings.ToLower(Name(tVar.Name())) + "s"
	}
	return s
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
 * 示例：insert into tableName (x,x) values (?, ?)
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
	var err error
	var sp *sql.Stmt
	if s.beginTx {
		sp, err = s.tx.Prepare(query)
	} else {
		sp, err = s.db.db.Prepare(query)
	}
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
 * InsertBatch
 * @Author：Jack-Z
 * @Description: 批量插入数据操作
 * 示例：insert into tableName (x,x) values (?, ?), (?, ?)
 * @receiver s
 * @param data
 * @return int64
 * @return int64
 * @return error
 */
func (s *GrSession) InsertBatch(data []any) (int64, int64, error) {
	if len(data) == 0 {
		return -1, -1, errors.New("no data to insert")
	}

	s.fieldNames(data[0])
	query := fmt.Sprintf("insert into %s (%s) values",
		s.tableName,
		strings.Join(s.fieldName, ","))

	var sb strings.Builder
	sb.WriteString(query)
	for index, _ := range data {
		sb.WriteString("(")
		sb.WriteString(strings.Join(s.placeHolder, ","))
		sb.WriteString(")")
		if index < len(data)-1 {
			sb.WriteString(",")
		}
	}
	s.batchValues(data)
	s.db.logger.Info(sb.String())

	var err error
	var sp *sql.Stmt
	if s.beginTx {
		sp, err = s.tx.Prepare(query)
	} else {
		sp, err = s.db.db.Prepare(query)
	}

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
 * Update
 * @Author：Jack-Z
 * @Description: update更新操作
 * 调用方式；db.New().Where("id", 1006).Where("age", 54).Update(user)
 * update user set age = 10 where id = 100;
 * @receiver s
 * @param data
 * @return int64
 * @return int64
 * @return error
 */
func (s *GrSession) Update(data ...any) (int64, int64, error) {
	if len(data) > 2 {
		return -1, -1, errors.New("param not valid")
	}
	if len(data) == 0 {
		query := fmt.Sprintf("update %s set %s", s.tableName, s.updateParam.String())
		var sb strings.Builder
		sb.WriteString(query)
		sb.WriteString(s.whereParam.String())
		s.db.logger.Info(sb.String())

		var err error
		var sp *sql.Stmt
		if s.beginTx {
			sp, err = s.tx.Prepare(query)
		} else {
			sp, err = s.db.db.Prepare(query)
		}

		if err != nil {
			return -1, -1, err
		}
		s.values = append(s.values, s.whereValues...)
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
	single := true
	if len(data) == 2 {
		single = false
	}
	if !single {
		if s.updateParam.String() != "" {
			s.updateParam.WriteString(",")
		}
		s.updateParam.WriteString(data[0].(string))
		s.updateParam.WriteString("= ?")
		s.values = append(s.values, data[1])
	} else {
		updateData := data[0]

		t := reflect.TypeOf(updateData)
		v := reflect.ValueOf(updateData)
		if t.Kind() != reflect.Pointer {
			panic(errors.New("updateData must be pointer"))
		}

		tVar := t.Elem()
		vVar := v.Elem()

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
			if s.updateParam.String() != "" {
				s.updateParam.WriteString(",")
			}
			s.updateParam.WriteString(sqlTag)
			s.updateParam.WriteString("= ?")
			s.values = append(s.values, vVar.Field(i).Interface())
		}
	}
	query := fmt.Sprintf("update %s set %s", s.tableName, s.updateParam.String())
	var sb strings.Builder
	sb.WriteString(query)
	sb.WriteString(s.whereParam.String())
	s.db.logger.Info(sb.String())
	var err error
	var sp *sql.Stmt
	if s.beginTx {
		sp, err = s.tx.Prepare(query)
	} else {
		sp, err = s.db.db.Prepare(query)
	}
	if err != nil {
		return -1, -1, err
	}
	s.values = append(s.values, s.whereValues...)
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
 * UpdateParam
 * @Author：Jack-Z
 * @Description: 支持"字段，值"的更新方式
 * 调用方式：db.New().Table("xxxx").Where("id", 10).UpdateParam("age", 34).Update()
 * @receiver s
 * @param field
 * @param value
 * @return *GrSession
 */
func (s *GrSession) UpdateParam(field string, value any) *GrSession {
	if s.updateParam.String() != "" {
		s.updateParam.WriteString(",")
	}
	s.updateParam.WriteString(field)
	s.updateParam.WriteString(" = ?")
	s.values = append(s.values, value)
	return s
}

/*
*
  - UpdateMap
  - @Author：Jack-Z
  - @Description: 支持map格式的更新值
  - 调用方式：db.
    New().Table("gr_user").
    Where("id", 2).
    UpdateMap(map[string]interface{}{
    "password": "iiiiiiii",
    }).
    Update()
  - @receiver s
  - @param data
  - @return *GrSession
*/
func (s *GrSession) UpdateMap(data map[string]any) *GrSession {
	for k, v := range data {
		if s.updateParam.String() != "" {
			s.updateParam.WriteString(",")
		}
		s.updateParam.WriteString(k)
		s.updateParam.WriteString(" = ?")
		s.values = append(s.values, v)
	}
	return s
}

/**
 * Where
 * @Author：Jack-Z
 * @Description: where 字段=值 条件处理
 * @receiver s
 * @param field 字段
 * @param value 值
 * @return *GrSession
 */
func (s *GrSession) Where(field string, value any) *GrSession {
	if s.whereParam.String() == "" {
		s.whereParam.WriteString(" where ")
	}
	s.whereParam.WriteString(field)
	s.whereParam.WriteString(" = ?")
	s.whereValues = append(s.whereValues, value)
	return s
}

/**
 * WhereMultiple
 * @Author：Jack-Z
 * @Description: where高级查询——兼容多种条件（比较、逻辑、模糊、范围等）
 * @receiver s
 * @param field
 * @param value
 * @return *GrSession
 */
func (s *GrSession) WhereMultiple(field string, value ...any) *GrSession {
	if s.whereParam.String() == "" {
		s.whereParam.WriteString(" where ")
	}
	s.whereParam.WriteString(field)
	s.whereValues = append(s.whereValues, value...)
	return s
}

/**
 * Like
 * @Author：Jack-Z
 * @Description: like模糊查询——like '%a%'
 * @receiver s
 * @param field
 * @param value
 * @return *GrSession
 */
func (s *GrSession) Like(field string, value any) *GrSession {
	if s.whereParam.String() == "" {
		s.whereParam.WriteString(" where ")
	}
	s.whereParam.WriteString(field)
	s.whereParam.WriteString(" like ? ")
	s.whereValues = append(s.whereValues, "%"+value.(string)+"%")
	return s
}

/**
 * LikeLeft
 * @Author：Jack-Z
 * @Description: like模糊查询——like '%a'
 * @receiver s
 * @param field
 * @param value
 * @return *GrSession
 */
func (s *GrSession) LikeLeft(field string, value any) *GrSession {
	if s.whereParam.String() == "" {
		s.whereParam.WriteString(" where ")
	}
	s.whereParam.WriteString(field)
	s.whereParam.WriteString(" like ? ")
	s.whereValues = append(s.whereValues, "%"+value.(string))
	return s
}

/**
 * LikeRight
 * @Author：Jack-Z
 * @Description: like模糊查询——like 'a%'
 * @receiver s
 * @param field
 * @param value
 * @return *GrSession
 */
func (s *GrSession) LikeRight(field string, value any) *GrSession {
	if s.whereParam.String() == "" {
		s.whereParam.WriteString(" where ")
	}
	s.whereParam.WriteString(field)
	s.whereParam.WriteString(" like ? ")
	s.whereValues = append(s.whereValues, value.(string)+"%")
	return s
}

/**
 * Group
 * @Author：Jack-Z
 * @Description: group by分组——group by a
 * @receiver s
 * @param field
 * @return *GrSession
 */
func (s *GrSession) Group(field ...string) *GrSession {
	s.whereParam.WriteString(" group by ")
	s.whereParam.WriteString(strings.Join(field, ","))
	return s
}

/**
 * OrderAsc
 * @Author：Jack-Z
 * @Description: order排序——order a asc
 * @receiver s
 * @param field
 * @return *GrSession
 */
func (s *GrSession) OrderAsc(field ...string) *GrSession {
	s.whereParam.WriteString(" order by ")
	s.whereParam.WriteString(strings.Join(field, ","))
	s.whereParam.WriteString(" asc ")
	return s
}

/**
 * OrderDesc
 * @Author：Jack-Z
 * @Description: order排序——order a,b desc
 * @receiver s
 * @param field
 * @return *GrSession
 */
func (s *GrSession) OrderDesc(field ...string) *GrSession {
	s.whereParam.WriteString(" order by ")
	s.whereParam.WriteString(strings.Join(field, ","))
	s.whereParam.WriteString(" desc ")
	return s
}

/**
 * Order
 * @Author：Jack-Z
 * @Description: order排序——order by a desc, b asc
 * @receiver s
 * @param fields
 * @return *GrSession
 */
func (s *GrSession) Order(fields ...string) *GrSession {
	size := len(fields)
	if size%2 != 0 {
		panic("order field must be even numbers")
	}
	s.whereParam.WriteString(" order by ")
	for i, v := range fields {
		s.whereParam.WriteString(v + " ")
		if i%2 != 0 && i < size-1 {
			s.whereParam.WriteString(",")
		}
	}
	return s
}

/**
 * Between
 * @Author：Jack-Z
 * @Description: where a between xx and xx
 * @receiver s
 * @param field
 * @param value
 * @return *GrSession
 */
func (s *GrSession) Between(field string, value ...any) *GrSession {
	if s.whereParam.String() == "" {
		s.whereParam.WriteString(" where ")
	}
	s.whereParam.WriteString(field)
	s.whereParam.WriteString(" between ? and ?")
	s.whereValues = append(s.whereValues, value...)
	return s
}

/**
 * Gt
 * @Author：Jack-Z
 * @Description: 大于（greater than）where a > xx
 * @receiver s
 * @param field
 * @param value
 * @return *GrSession
 */
func (s *GrSession) Gt(field string, value any) *GrSession {
	if s.whereParam.String() == "" {
		s.whereParam.WriteString(" where ")
	}
	s.whereParam.WriteString(field)
	s.whereParam.WriteString(" > ? ")
	s.whereValues = append(s.whereValues, value)
	return s
}

/**
 * Ge
 * @Author：Jack-Z
 * @Description: 大于等于（Great and Equal）where a >= xx
 * @receiver s
 * @param field
 * @param value
 * @return *GrSession
 */
func (s *GrSession) Ge(field string, value any) *GrSession {
	if s.whereParam.String() == "" {
		s.whereParam.WriteString(" where ")
	}
	s.whereParam.WriteString(field)
	s.whereParam.WriteString(" >= ? ")
	s.whereValues = append(s.whereValues, value)
	return s
}

/**
 * Lt
 * @Author：Jack-Z
 * @Description: 小于（greater than）where a < xx
 * @receiver s
 * @param field
 * @param value
 * @return *GrSession
 */
func (s *GrSession) Lt(field string, value any) *GrSession {
	if s.whereParam.String() == "" {
		s.whereParam.WriteString(" where ")
	}
	s.whereParam.WriteString(field)
	s.whereParam.WriteString(" < ? ")
	s.whereValues = append(s.whereValues, value)
	return s
}

/**
 * Le
 * @Author：Jack-Z
 * @Description: 小于等于（Less than or equal）where a <= xx
 * @receiver s
 * @param field
 * @param value
 * @return *GrSession
 */
func (s *GrSession) Le(field string, value any) *GrSession {
	if s.whereParam.String() == "" {
		s.whereParam.WriteString(" where ")
	}
	s.whereParam.WriteString(field)
	s.whereParam.WriteString(" <= ? ")
	s.whereValues = append(s.whereValues, value)
	return s
}

/**
 * Le
 * @Author：Jack-Z
 * @Description: NE(Not Equal to)不等于——where a <> xxx
 * @receiver s
 * @param field
 * @param value
 * @return *GrSession
 */
func (s *GrSession) Ne(field string, value any) *GrSession {
	if s.whereParam.String() == "" {
		s.whereParam.WriteString(" where ")
	}
	s.whereParam.WriteString(field)
	s.whereParam.WriteString(" <> ? ")
	s.whereValues = append(s.whereValues, value)
	return s
}

/**
 * In
 * @Author：Jack-Z
 * @Description: where a in (1,2,3)
 * @receiver s
 * @param field
 * @param values
 * @return *GrSession
 */
func (s *GrSession) In(field string, values ...any) *GrSession {
	size := len(values)
	if size <= 0 {
		panic("parameter numbers must be greater than 0")
	}
	if s.whereParam.String() == "" {
		s.whereParam.WriteString(" where ")
	}
	s.whereParam.WriteString(field)
	s.whereParam.WriteString(" in ( ")

	for i := 0; i < size; i++ {
		s.whereParam.WriteString("?")
		if i < size-1 {
			s.whereParam.WriteString(", ")
		}
	}
	s.whereParam.WriteString(" ) ")
	s.whereValues = append(s.whereValues, values...)
	return s
}

/**
 * NotIn
 * @Author：Jack-Z
 * @Description: where a not in (1,2,3)
 * @receiver s
 * @param field
 * @param values
 * @return *GrSession
 */
func (s *GrSession) NotIn(field string, values ...any) *GrSession {
	size := len(values)
	if size <= 0 {
		panic("parameter numbers must be greater than 0")
	}
	if s.whereParam.String() == "" {
		s.whereParam.WriteString(" where ")
	}
	s.whereParam.WriteString(field)
	s.whereParam.WriteString(" not in ( ")

	for i := 0; i < size; i++ {
		s.whereParam.WriteString("?")
		if i < size-1 {
			s.whereParam.WriteString(", ")
		}
	}
	s.whereParam.WriteString(" ) ")
	s.whereValues = append(s.whereValues, values...)
	return s
}

/**
 * IsNull
 * @Author：Jack-Z
 * @Description: where a is null
 * @receiver s
 * @param field
 * @return *GrSession
 */
func (s *GrSession) IsNull(field string) *GrSession {
	if s.whereParam.String() == "" {
		s.whereParam.WriteString(" where ")
	}
	s.whereParam.WriteString(field)
	s.whereParam.WriteString(" is null ")
	return s
}

/**
 * IsNotNull
 * @Author：Jack-Z
 * @Description: where a is not null
 * @receiver s
 * @param field
 * @return *GrSession
 */
func (s *GrSession) IsNotNull(field string) *GrSession {
	if s.whereParam.String() == "" {
		s.whereParam.WriteString(" where ")
	}
	s.whereParam.WriteString(field)
	s.whereParam.WriteString(" is not null ")
	return s
}

/**
 * Count
 * @Author：Jack-Z
 * @Description: 聚合函数——count
 * @receiver s
 * @param fields
 * @return int64
 * @return error
 */
func (s *GrSession) Count(fields ...string) (int64, error) {
	if len(fields) == 0 {
		return s.Aggregate("count", "*")
	}
	return s.Aggregate("count", fields[0])
}

/**
 * Aggregate
 * @Author：Jack-Z
 * @Description: 聚合函数公共方法
 * @receiver s
 * @param funcName  方法名
 * @param field   字段名
 * @return int64
 * @return error
 */
func (s *GrSession) Aggregate(funcName, field string) (int64, error) {
	var aggSb strings.Builder
	aggSb.WriteString(funcName)
	aggSb.WriteString("(")
	aggSb.WriteString(field)
	aggSb.WriteString(")")
	query := fmt.Sprintf("select %s from %s ", aggSb.String(), s.tableName)
	var sb strings.Builder
	sb.WriteString(query)
	sb.WriteString(s.whereParam.String())
	s.db.logger.Info(sb.String())

	prepare, err := s.db.db.Prepare(sb.String())
	if err != nil {
		return 0, err
	}
	row := prepare.QueryRow(s.whereValues...)
	if row.Err() != nil {
		return 0, err
	}
	var c int64
	err = row.Scan(&c)
	if err != nil {
		return 0, err
	}
	return c, nil
}

/**
 * And
 * @Author：Jack-Z
 * @Description: where 字段=值 and 字段1=值1 条件处理
 * @receiver s
 * @return *GrSession
 */
func (s *GrSession) And() *GrSession {
	s.whereParam.WriteString(" and ")
	return s
}

/**
 * Or
 * @Author：Jack-Z
 * @Description: where 字段=值 or 字段=值
 * @receiver s
 * @return *GrSession
 */
func (s *GrSession) Or() *GrSession {
	s.whereParam.WriteString(" or ")
	return s
}

/**
 * QueryExec
 * @Author：Jack-Z
 * @Description: 原生sql——insert/update/delete
 * @receiver s
 * @param sql
 * @param values
 * @return int64
 * @return error
 */
func (s *GrSession) QueryExec(query string, values ...any) (int64, error) {
	var err error
	var prepare *sql.Stmt
	if s.beginTx {
		prepare, err = s.tx.Prepare(query)
	} else {
		prepare, err = s.db.db.Prepare(query)
	}

	if err != nil {
		return 0, err
	}
	exec, err := prepare.Exec(values)
	if err != nil {
		return 0, err
	}
	if strings.Contains(strings.ToLower(query), "insert") {
		return exec.LastInsertId()
	}
	return exec.RowsAffected()
}

/**
 * QueryRow
 * @Author：Jack-Z
 * @Description: 原生sql——select
 * @receiver s
 * @param sql
 * @param data
 * @param queryValues
 * @return error
 */
func (s *GrSession) QueryRow(sql string, data any, queryValues ...any) error {
	t := reflect.TypeOf(data)
	if t.Kind() != reflect.Pointer {
		return errors.New("data must be a pointer")
	}
	stmt, err := s.db.db.Prepare(sql)
	if err != nil {
		return err
	}
	rows, err := stmt.Query(queryValues...)
	if err != nil {
		return err
	}

	columns, err := rows.Columns()
	if err != nil {
		return err
	}
	values := make([]any, len(columns))
	fieldsScan := make([]any, len(columns))
	for i := range fieldsScan {
		fieldsScan[i] = &values[i]
	}

	if rows.Next() {
		err := rows.Scan(fieldsScan...)
		if err != nil {
			return err
		}
		tVar := t.Elem()
		vVar := reflect.ValueOf(data).Elem()
		for i := 0; i < tVar.NumField(); i++ {
			name := tVar.Field(i).Name
			tag := tVar.Field(i).Tag
			sqlTag := tag.Get("grorm")
			if sqlTag == "" {
				sqlTag = strings.ToLower(Name(name))
			} else {
				if strings.Contains(sqlTag, ",") {
					sqlTag = sqlTag[:strings.Index(sqlTag, ",")]
				}
			}

			for j, colName := range columns {
				if sqlTag == colName {
					target := values[j]
					targetVal := reflect.ValueOf(target)
					fieldType := tVar.Field(i).Type
					// 	类型转换
					result := reflect.ValueOf(targetVal.Interface()).Convert(fieldType)
					vVar.Field(i).Set(result)
				}
			}
		}
	}
	return nil
}

/**
 * Begin
 * @Author：Jack-Z
 * @Description: 事务开启
 * @receiver s
 * @return error
 */
func (s *GrSession) Begin() error {
	begin, err := s.db.db.Begin()
	if err != nil {
		return err
	}
	s.tx = begin
	s.beginTx = true
	return nil
}

/**
 * Commit
 * @Author：Jack-Z
 * @Description: 事务提交
 * @receiver s
 * @return error
 */
func (s *GrSession) Commit() error {
	err := s.tx.Commit()
	if err != nil {
		return err
	}
	s.beginTx = false
	return nil
}

/**
 * Rollback
 * @Author：Jack-Z
 * @Description: 事务回滚
 * @receiver s
 * @return error
 */
func (s *GrSession) Rollback() error {
	err := s.tx.Rollback()
	if err != nil {
		return err
	}
	s.beginTx = false
	return nil
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
		s.tableName = s.db.Prefix + strings.ToLower(Name(tVar.Name())) + "s"
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
 * batchValues
 * @Author：Jack-Z
 * @Description: 多条数据的value处理
 * @receiver s
 * @param data
 */
func (s *GrSession) batchValues(data []any) {
	s.values = make([]any, 0)
	for _, v := range data {
		t := reflect.TypeOf(v)
		v := reflect.ValueOf(v)
		if t.Kind() != reflect.Pointer {
			panic(errors.New("data must be pointer"))
		}

		tVar := t.Elem()
		vVar := v.Elem()
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
			}
			id := vVar.Field(i).Interface()
			// 对id做个判断，如果其值小于等于0，数据库可能是自增，跳过此字段
			if strings.ToLower(sqlTag) == "id" && IsAutoId(id) {
				continue
			}
			s.values = append(s.values, vVar.Field(i).Interface())
		}
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

/**
 * SelectOne
 * @Author：Jack-Z
 * @Description: selelct——查询数据（单条）
 * @receiver s
 * @param data 查询结果的映射数据
 * @param fields 查询的字段
 * @return error
 */
func (s *GrSession) SelectOne(data any, fields ...string) error {
	t := reflect.TypeOf(data)
	if t.Kind() != reflect.Pointer {
		return errors.New("data must be a pointer")
	}

	fieldsStr := "*"
	if len(fields) > 0 {
		fieldsStr = strings.Join(fields, ",")
	}
	query := fmt.Sprintf("select %s from %s", fieldsStr, s.tableName)
	var sb strings.Builder
	sb.WriteString(query)
	sb.WriteString(s.whereParam.String())
	s.db.logger.Info(sb.String())

	prepare, err := s.db.db.Prepare(sb.String())
	if err != nil {
		return err
	}
	rows, err := prepare.Query(s.whereValues...)
	columns, err := rows.Columns()
	if err != nil {
		return err
	}
	values := make([]any, len(columns))
	fieldsScan := make([]any, len(columns))
	for i := range fieldsScan {
		fieldsScan[i] = &values[i]
	}

	if rows.Next() {
		err := rows.Scan(fieldsScan...)
		if err != nil {
			return err
		}
		tVar := t.Elem()
		vVar := reflect.ValueOf(data).Elem()
		for i := 0; i < tVar.NumField(); i++ {
			name := tVar.Field(i).Name
			tag := tVar.Field(i).Tag
			sqlTag := tag.Get("grorm")
			if sqlTag == "" {
				sqlTag = strings.ToLower(Name(name))
			} else {
				if strings.Contains(sqlTag, ",") {
					sqlTag = sqlTag[:strings.Index(sqlTag, ",")]
				}
			}

			for j, colName := range columns {
				if sqlTag == colName {
					target := values[j]
					targetVal := reflect.ValueOf(target)
					fieldType := tVar.Field(i).Type
					// 	类型转换
					result := reflect.ValueOf(targetVal.Interface()).Convert(fieldType)
					vVar.Field(i).Set(result)
				}
			}
		}
	}
	return nil
}

/**
 * Select
 * @Author：Jack-Z
 * @Description: select--查询多条数据
 * @receiver s
 * @param data
 * @param fields
 * @return []any
 * @return error
 */
func (s *GrSession) Select(data any, fields ...string) ([]any, error) {
	t := reflect.TypeOf(data)
	if t.Kind() != reflect.Pointer {
		return nil, errors.New("data must be a pointer")
	}

	fieldsStr := "*"
	if len(fields) > 0 {
		fieldsStr = strings.Join(fields, ",")
	}
	query := fmt.Sprintf("select %s from %s ", fieldsStr, s.tableName)
	var sb strings.Builder
	sb.WriteString(query)
	sb.WriteString(s.whereParam.String())
	s.db.logger.Info(sb.String())

	prepare, err := s.db.db.Prepare(sb.String())
	if err != nil {
		return nil, err
	}
	rows, err := prepare.Query(s.whereValues...)
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	result := make([]any, 0)
	for {
		if rows.Next() {
			data := reflect.New(t.Elem()).Interface()
			values := make([]any, len(columns))
			fieldsScan := make([]any, len(columns))
			for i := range fieldsScan {
				fieldsScan[i] = &values[i]
			}

			err := rows.Scan(fieldsScan...)
			if err != nil {
				return nil, err
			}
			tVar := t.Elem()
			vVar := reflect.ValueOf(data).Elem()
			for i := 0; i < tVar.NumField(); i++ {
				name := tVar.Field(i).Name
				tag := tVar.Field(i).Tag
				sqlTag := tag.Get("grorm")
				if sqlTag == "" {
					sqlTag = strings.ToLower(Name(name))
				} else {
					if strings.Contains(sqlTag, ",") {
						sqlTag = sqlTag[:strings.Index(sqlTag, ",")]
					}
				}

				for j, colName := range columns {
					if sqlTag == colName {
						target := values[j]
						targetVal := reflect.ValueOf(target)
						fieldType := tVar.Field(i).Type
						// 	类型转换
						result := reflect.ValueOf(targetVal.Interface()).Convert(fieldType)
						vVar.Field(i).Set(result)
					}
				}
			}
			result = append(result, data)
		} else {
			break
		}
	}
	return result, nil
}

/**
 * Delete
 * @Author：Jack-Z
 * @Description: delete——删除数据
 * @receiver s
 * @return int64
 * @return error
 */
func (s *GrSession) Delete() (int64, error) {
	query := fmt.Sprintf("delete from %s ", s.tableName)
	var sb strings.Builder
	sb.WriteString(query)
	sb.WriteString(s.whereParam.String())
	s.db.logger.Info(sb.String())
	var err error
	var prepare *sql.Stmt
	if s.beginTx {
		prepare, err = s.tx.Prepare(query)
	} else {
		prepare, err = s.db.db.Prepare(query)
	}
	if err != nil {
		return 0, err
	}
	exec, err := prepare.Exec(s.whereValues...)
	if err != nil {
		return 0, err
	}
	return exec.RowsAffected()
}

// 解析结构体字段以获取列定义
func columnsFromStruct(model any) ([]string, error) {
	var columns []string
	t := reflect.TypeOf(model)

	tVar := t.Elem()
	dbNameNewParts := []string{}
	dbNameIndexParts := []string{}
	for i := 0; i < tVar.NumField(); i++ {
		field := tVar.Field(i)
		// 列名
		fieldName := field.Tag.Get("json")
		if fieldName == "" {
			continue
		}

		// 列的约束条件
		dbName := field.Tag.Get("grorm")
		dbNameParts := strings.Split(strings.ToLower(dbName), ",")
		if dbName == "" { // 如果没有db标签或db标签为空，就根据结构体定义的类型给默认设置
			if field.Type.Kind() == reflect.Int ||
				field.Type.Kind() == reflect.Int64 ||
				field.Type.Kind() == reflect.Uint ||
				field.Type.Kind() == reflect.Uint8 ||
				field.Type.Kind() == reflect.Uint32 ||
				field.Type.Kind() == reflect.Uint64 ||
				field.Type.Kind() == reflect.Int32 {
				dbNameNewParts = append(dbNameNewParts, fmt.Sprintf("`%v` int not null default 0", fieldName))
			} else if field.Type.Kind() == reflect.Bool {
				dbNameNewParts = append(dbNameNewParts, fmt.Sprintf("`%v` tinyint(1) not null default 0", fieldName))
			} else if field.Type.Kind() == reflect.Float64 || field.Type.Kind() == reflect.Float32 {
				dbNameNewParts = append(dbNameNewParts, fmt.Sprintf("`%v` decimal(6,2) not null default 0", fieldName))
			} else if field.Type == reflect.TypeOf(time.Time{}) {
				dbNameNewParts = append(dbNameNewParts, fmt.Sprintf("`%v` datetime not null default '0000-00-00 00:00:00'", fieldName))
			} else {
				dbNameNewParts = append(dbNameNewParts, fmt.Sprintf("`%v` varchar(100) not null default ''", fieldName))
			}
		} else if len(dbNameParts) > 0 {
			sqlArr := []string{fmt.Sprintf("`%v`", fieldName)}
			for _, v := range dbNameParts {
				vSplit := strings.Split(v, ":")
				if len(vSplit) > 1 {
					switch vSplit[0] {
					case "type":
						sqlArr = append(sqlArr, vSplit[1])
					case "default":
						sqlArr = append(sqlArr, fmt.Sprintf("default %v", vSplit[1]))
					case "index":
						dbNameIndexParts = append(dbNameIndexParts, fmt.Sprintf("index %v_idx (%v)", fieldName, fieldName))
					case "unique_index":
						dbNameIndexParts = append(dbNameIndexParts, fmt.Sprintf("unique %v_idx (%v)", fieldName, fieldName))
					case "fulltext_index":
						dbNameIndexParts = append(dbNameIndexParts, fmt.Sprintf("fulltext %v_idx (%v)", fieldName, fieldName))
					}
				} else {
					sqlArr = append(sqlArr, v)
				}
			}
			dbNameNewParts = append(dbNameNewParts, strings.Join(sqlArr, " "))
		}
	}
	columns = append(columns, dbNameNewParts...)
	columns = append(columns, dbNameIndexParts...)
	return columns, nil
}

// 构建CREATE TABLE语句并创建
func (db *GrDb) AutoMigrateMySQL(model any) error {
	t := reflect.TypeOf(model)
	tableName := strings.ToLower(Name(t.Elem().Name())) + "s" // 获取表名字
	columns, err := columnsFromStruct(model)
	if err != nil {
		return err
	}

	// 构建CREATE TABLE语句
	columnDefs := strings.Join(columns, ", ")
	createTableStmt := fmt.Sprintf("CREATE TABLE IF NOT EXISTS `%s` (%s);", tableName, columnDefs)
	fmt.Println(createTableStmt)

	// 执行建表语句
	sqlPer, err := db.db.Prepare(createTableStmt)
	if err != nil {
		return err
	}
	exec, err := sqlPer.Exec()
	if err != nil {
		return err
	}
	_, err = exec.RowsAffected()
	if err != nil {
		return err
	}

	return nil
}
