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
		s.tableName = s.db.Prefix + strings.ToLower(Name(tVar.Name()))
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
	sp, err := s.db.db.Prepare(sb.String())
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

		sp, err := s.db.db.Prepare(sb.String())
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

	sp, err := s.db.db.Prepare(sb.String())
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
 * @Description: 支持"字段， 值"的更新方式
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
	query := fmt.Sprintf("select %s from %s", fieldsStr, s.tableName)
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

	prepare, err := s.db.db.Prepare(sb.String())
	if err != nil {
		return 0, err
	}
	exec, err := prepare.Exec(s.whereValues...)
	if err != nil {
		return 0, err
	}
	return exec.RowsAffected()
}
