package Wsy

import (
	"fmt"
	"time"
	"sort"
	"errors"
	"strings"
	"database/sql"
	"encoding/json"
	_ "github.com/go-sql-driver/mysql"
	
)


/*
手动配置DB方法
var MainDB = Wsy.DBConfig{
    Type:     "mysql",
    Host:     "127.0.0.1",
    Port:     3306,
    Username: "wsyos",
    Password: "Aa@linqijing2025",
    Database: "wsyos",
    Charset:  "utf8mb4",
	MaxOpenConns: 200,
    MaxIdleConns: 100,
}
Wsy.DB.Config(MainDB)
*/

// WsyDB 提供了轻量级的 MySQL 数据库操作封装
type WsyDB struct {
	db        *sql.DB // 内部数据库连接
	dsn       string  // 数据库连接字符串
	config    DBConfig // 保存全局配置
	Page      int
	Limit     int      // 全局分页条数，默认20
	Debug     bool     // 是否开启SQL调试
}



// 移除单例相关变量

// DBConfig MySQL配置结构体
type DBConfig struct {
	Type         string // 数据库类型: mysql, sqlite, mssql
	Host         string // 数据库主机地址
	Port         int    // 数据库端口
	UserName     string // 数据库用户名
	PassWord     string // 数据库密码
	Database     string // 数据库名称
	Charset      string // 字符集，推荐使用utf8mb4
	MaxOpenConns int    // 最大打开连接数
	MaxIdleConns int    // 最大空闲连接数
}



// 设置全局配置
func (m *WsyDB) Config(cfg DBConfig) {
    m.config = cfg
}

// Open 检查数据库连接是否可用，不可用则自动Init
func (m *WsyDB) Open() error {
	if m.db == nil {
		return m.Init()
	}
	if err := m.db.Ping(); err != nil {
		return m.Init()
	}
	return nil
}


// Init 初始化数据库连接，只用全局配置
func (m *WsyDB) Init() error {
	var dbConf DBConfig
	if m.config != (DBConfig{}) {
		dbConf = m.config
	} else {
		Logs("INFO","DB","数据库全局配置未设置，请先调用Config方法")
		return errors.New("数据库全局配置未设置，请先调用Config方法")
	}

	m.dsn = fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=true&loc=Local",
		dbConf.UserName,
		dbConf.PassWord,
		dbConf.Host,
		dbConf.Port,
		dbConf.Database,
		dbConf.Charset,
	)
	// 创建数据库连接
	db, err := sql.Open("mysql", m.dsn)
	if err != nil {
		Logs("Error", "DB", "数据库连接失败，请检查配置文件,%v" + err.Error())
		return errors.New("数据库连接失败: " + err.Error())
	}

	// 设置连接池参数
	if dbConf.MaxOpenConns > 0 {
		db.SetMaxOpenConns(dbConf.MaxOpenConns)
	} else {
		db.SetMaxOpenConns(200)
	}
	if dbConf.MaxIdleConns > 0 {
		db.SetMaxIdleConns(dbConf.MaxIdleConns)
	} else {
		db.SetMaxIdleConns(100)
	}
	db.SetConnMaxLifetime(time.Hour)

	// 测试连接是否有效
	if err := db.Ping(); err != nil {
		db.Close() // 关闭无效连接
		Logs("Error", "DB", "数据库连接失败，请检查配置文件, "+err.Error())
		return errors.New("数据库ping失败: " + err.Error())
	}

	m.db = db
	return nil
}
// LoadConfig("/opt/config.ini", "dbkey", "Data")
// LoadConfig("/opt/config.ini", "db")
// LoadConfigFromEncryptedIni 从加密的ini配置读取并解密为DBConfig
func (m *WsyDB) LoadInit(iniPath ...string) (DBConfig, error) {
	var cfg DBConfig
	var mconf map[string]string
	var jsonStr string
	configPath := Set.File
	if len(iniPath) > 0 && iniPath[0] != "" {
		configPath = iniPath[0]
	}
	// 设置默认 section 为 "Database"
	sectionName := "Database"
	if len(iniPath) > 1 && iniPath[1] != "" {
		sectionName = iniPath[1]
	}
	enc := Fso.ReadIni(configPath, sectionName, "KeyData")
	if enc != "" {
		jsonStr = Key.DeCode(enc)
		if jsonStr == "" {
			return cfg, errors.New("数据库配置解密失败")
		}
	} else {
		jsonStr = Fso.ReadIni(configPath, sectionName)
		if jsonStr == "" {
			return cfg, errors.New("未找到数据库配置")
		}
	}
	if err := json.Unmarshal([]byte(jsonStr), &mconf); err != nil {
		return cfg, errors.New("数据库配置反序列化失败: " + err.Error())
	}

	cfg = DBConfig{
		Type:         mconf["type"],
		Host:         mconf["host"],
		Port:         Str.ToInt(mconf["port"]),
		UserName:     mconf["username"],
		PassWord:     mconf["password"],
		Database:     mconf["database"],
		Charset:      mconf["charset"],
		MaxOpenConns: Str.ToInt(mconf["maxopenconns"]),
		MaxIdleConns: Str.ToInt(mconf["maxidleconns"]),
	}
	m.config = cfg
	return cfg, nil
}

// GetFullSQL 生成包含参数值的完整SQL语句
func (m *WsyDB) ToSQL(query string, args ...interface{}) string {
	var b strings.Builder
	argIndex := 0
	for i := 0; i < len(query); i++ {
		if query[i] == '?' && argIndex < len(args) {
			val := args[argIndex]
			switch v := val.(type) {
			case string:
				b.WriteString("'" + strings.ReplaceAll(v, "'", "\\'") + "'")
			case nil:
				b.WriteString("NULL")
			default:
				b.WriteString(fmt.Sprintf("%v", v))
			}
			argIndex++
		} else {
			b.WriteByte(query[i])
		}
	}
	return b.String()
}

// Query 执行SQL查询并返回*sql.Rows结果集
// 这是一个底层方法，允许直接访问数据库驱动的查询结果
// 注意：调用者需要负责关闭返回的*sql.Rows以避免资源泄漏
func (m *WsyDB) Query(query string, args ...interface{}) (*sql.Rows, error) {
	// 确保数据库连接已初始化
	if m.db == nil {
		return nil, errors.New("数据库连接未初始化，请先调用Open方法")
	}
	if m.Debug {
		Echo(m.ToSQL(query, args...))
	}
	rows, err := m.db.Query(query, args...)
	if err != nil {
		return nil, errors.New("执行查询失败: " + err.Error())
	}
	return rows, nil
}


// Exec 执行不返回行的SQL语句（如INSERT, UPDATE, DELETE）
// 返回SQL结果摘要和可能的错误
func (m *WsyDB) Exec(query string, args ...interface{}) (sql.Result, error) {
	// 确保数据库连接已初始化
	if m.db == nil {
		return nil, errors.New("数据库连接未初始化，请先调用Open方法")
	}
	if m.Debug {
		Logs("SQL", "Exec", m.ToSQL(query, args...))
	}
	result, err := m.db.Exec(query, args...)
	if err != nil {
		return nil, errors.New("执行语句失败: " + err.Error())
	}

	return result, nil
}


// Sel 执行SQL查询并返回单行结果（只取第一行）
func (m *WsyDB) Sel(query string, args ...interface{}) (map[string]interface{}, error) {
	if m.db == nil {
		return nil, errors.New("数据库未连接,请先调用Open方法")
	}
	if m.Debug {
		Echo(m.ToSQL(query, args...))
	}
	rows, err := m.db.Query(query, args...)
	if err != nil {
		return nil, errors.New("执行查询失败: " + err.Error())
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, errors.New(err.Error())
	}
	if !rows.Next() {
		return nil, nil
	}
	values := make([]interface{}, len(columns))
	valuePtrs := make([]interface{}, len(columns))
	for i := range columns {
		valuePtrs[i] = &values[i]
	}
	if err := rows.Scan(valuePtrs...); err != nil {
		return nil, errors.New(err.Error())
	}
	result := make(map[string]interface{})
	for i, col := range columns {
		val := values[i]
		if b, ok := val.([]byte); ok {
			result[col] = string(b)
		} else {
			result[col] = val
		}
	}
	return result, nil
}
// Sels 执行多行查询并返回结果集
func (m *WsyDB) Sels(query string, args ...interface{}) ([]map[string]interface{}, error) {
	if m.db == nil {
		return nil, errors.New("数据库未连接,请先调用Open方法")
	}
	if m.Debug {
		Echo(m.ToSQL(query, args...))
	}
	rows, err := m.db.Query(query, args...)
	if err != nil {
		return nil, errors.New("执行查询失败: " + err.Error())
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, errors.New("获取结果列信息失败: " + err.Error())
	}

	results := make([]map[string]interface{}, 0)
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range columns {
			valuePtrs[i] = &values[i]
		}
		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, errors.New("扫描结果行失败: " + err.Error())
		}
		row := make(map[string]interface{}, len(columns))
		for i, col := range columns {
			val := values[i]
			if b, ok := val.([]byte); ok {
				row[col] = string(b)
			} else {
				row[col] = val
			}
		}
		results = append(results, row)
	}
	if err = rows.Err(); err != nil {
		return nil, errors.New("遍历结果集时发生错误: " + err.Error())
	}
	return results, nil
}
// List 执行不带分页的查询，返回完整结果集
// 参数：
//   - tables: 表名数组，支持多表联接
//   - columns: 要查询的列名数组
//   - where: WHERE 条件子句（不含 WHERE 关键字）
//   - args: WHERE 条件中的参数值
//   - value: 可选参数，依次为：
//       - 排序子句（不含 ORDER BY 关键字）
//       - 分组字段（不含 GROUP BY 关键字）
//
// 返回：
//   - []map[string]interface{}: 查询结果集
//   - error: 错误信息
func (m *WsyDB) SelRows(tables []string, columns []string, where string, args []interface{}, value ...interface{}) ([]map[string]interface{}, error) {
	var sqlGroup, orderBy string
	if len(value) > 0 {
		orderBy = Str.ToString(value[0])
	}
	if len(value) > 1 && Str.ToString(value[1]) != "" {
		sqlGroup = "GROUP BY " + Str.ToString(value[1])
	}
	sqlWhere := Str.IIF(where != "", "WHERE "+where, "")
	sqlOrder := Str.IIF(orderBy != "", "ORDER BY "+orderBy, "")
	sqlTable := strings.Join(tables, " ")
	sqlColumns := strings.Join(columns, ", ")

	query := fmt.Sprintf("SELECT %s FROM %s %s %s %s", sqlColumns, sqlTable, sqlWhere, sqlGroup, sqlOrder)
	items, err := m.Sels(query, args...)
	if err != nil {
		return nil, errors.New("查询数据失败: " + err.Error())
	}
	return items, nil
}
// Update 执行更新操作
// 参数：
//   - tableName: 要更新的表名
//   - data: 要更新的字段和值的映射
//   - where: WHERE条件子句（不含WHERE关键字）
//   - args: WHERE条件中的参数值
//
// 返回：
//   - rowsAffected: 受影响的行数
//   - error: 错误信息
func (m *WsyDB) Update(tableName string, data map[string]interface{}, where string, args ...interface{}) (int64, error) {
	if m.db == nil {
		return 0, errors.New("数据库连接未初始化，请先调用Open方法")
	}
	if tableName == "" {
		return 0, errors.New("表名不能为空")
	}
	if len(data) == 0 {
		return 0, errors.New("更新数据不能为空！")
	}
	var setClauses []string
	var setArgs []interface{}
	for field, value := range data {
		setClauses = append(setClauses, fmt.Sprintf("%s = ?", field))
		setArgs = append(setArgs, value)
	}
	query := fmt.Sprintf("UPDATE %s SET %s", tableName, strings.Join(setClauses, ", "))
	if where != "" {
		query += " WHERE " + where
	}
	allArgs := append(setArgs, args...)

	if m.Debug {
		Echo(m.ToSQL(query, allArgs...))
	}
	result, err := m.db.Exec(query, allArgs...)
	if err != nil {
		return 0, errors.New("执行更新失败: " + err.Error())
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, errors.New("获取受影响行数失败: " + err.Error())
	}
	return rowsAffected, nil
}

// UpdateBatch 批量更新多条记录
// 参数：
//   - tableName: 要更新的表名
//   - data: 要更新的字段和值的映射数组，每个元素代表一条记录
//   - keyField: 用于标识每条记录的键字段（支持单字段或逗号分隔的多字段）
//   - where: 附加WHERE条件（可选，不含WHERE关键字）
//   - args: 附加WHERE条件中的参数值
//
// 返回：
//   - rowsAffected: 受影响的行数
//   - error: 错误信息
func (m *WsyDB) UpdateBatch(tableName string, data []map[string]interface{}, keyField string, where string, args ...interface{}) (int64, error) {
	if m.db == nil {
		return 0, errors.New("数据库连接未初始化，请先调用Open方法")
	}
	if tableName == "" {
		return 0, errors.New("表名不能为空")
	}
	if len(data) == 0 {
		return 0, errors.New("更新数据不能为空")
	}
	if keyField == "" {
		return 0, errors.New("键字段不能为空")
	}
	keyFields := strings.Split(keyField, ",")
	for i := range keyFields {
		keyFields[i] = strings.TrimSpace(keyFields[i])
	}
	tx, err := m.db.Begin()
	if err != nil {
		return 0, errors.New("开始事务失败: " + err.Error())
	}
	var totalAffected int64
	for _, item := range data {
		var keyValues []interface{}
		for _, kf := range keyFields {
			v, ok := item[kf]
			if !ok {
				tx.Rollback()
				return 0, errors.New("记录缺少键字段 %s" + err.Error())
			}
			keyValues = append(keyValues, v)
		}
		updateData := make(map[string]interface{})
		for k, v := range item {
			found := false
			for _, kf := range keyFields {
				if k == kf {
					found = true
					break
				}
			}
			if !found {
				updateData[k] = v
			}
		}
		if len(updateData) == 0 {
			continue // 跳过只有键字段的记录
		}
		var whereParts []string
		for _, kf := range keyFields {
			whereParts = append(whereParts, kf+" = ?")
		}
		itemWhere := strings.Join(whereParts, " AND ")
		itemArgs := keyValues
		if where != "" {
			itemWhere += " AND (" + where + ")"
			itemArgs = append(itemArgs, args...)
		}
		var setClauses []string
		var setArgs []interface{}
		for field, value := range updateData {
			setClauses = append(setClauses, fmt.Sprintf("%s = ?", field))
			setArgs = append(setArgs, value)
		}
		query := fmt.Sprintf("UPDATE %s SET %s WHERE %s",tableName,strings.Join(setClauses, ", "),itemWhere)
		allArgs := append(setArgs, itemArgs...)
		result, err := tx.Exec(query, allArgs...)
		if err != nil {
			tx.Rollback()
			return 0, errors.New("执行批量更新失败: " + err.Error())
		}
		// 累计受影响的行数
		affected, err := result.RowsAffected()
		if err != nil {
			tx.Rollback()
			return 0, errors.New("获取受影响行数失败: " + err.Error())
		}
		totalAffected += affected
	}
	// 提交事务
	if err := tx.Commit(); err != nil {
		return 0, errors.New("提交事务失败: " + err.Error())
	}
	return totalAffected, nil
}


// Insert 使用字段映射插入单条记录
// 参数：
//   - tableName: 要插入的表名
//   - data: 字段名到值的映射
//
// 返回：
//   - lastInsertID: 自增主键ID（如果有）
//   - error: 错误信息
func (m *WsyDB) Insert(tableName string, data map[string]interface{}) (int64, error) {
	// 确保数据库连接已初始化
	if m.db == nil {
		return 0, fmt.Errorf("数据库连接未初始化，请先调用Open方法")
	}
	if tableName == "" {
		return 0, fmt.Errorf("表名不能为空")
	}
	if len(data) == 0 {
		return 0, fmt.Errorf("插入数据不能为空")
	}
	// 构建列名和参数占位符
	var columns []string
	var placeholders []string
	var values []interface{}

	for col, val := range data {
		columns = append(columns, col)
		placeholders = append(placeholders, "?")
		values = append(values, val)
	}

	// 构建INSERT语句
	query := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s)",
		tableName,
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "),
	)
	result, err := m.db.Exec(query, values...)
	if err != nil {
		return 0, fmt.Errorf("执行插入失败: %v", err)
	}

	// 获取最后插入的ID
	lastID, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("获取最后插入ID失败: %v", err)
	}

	return lastID, nil
}

// InsertBatch 批量插入多条记录
// 参数：
//   - tableName: 要插入的表名
//   - dataList: 要插入的数据列表，每个元素是一个字段到值的映射
//
// 返回：
//   - affectedRows: 插入的记录数
//   - error: 错误信息
func (m *WsyDB) InsertBatch(tableName string, dataList []map[string]interface{}) (int64, error) {
	// 验证基本参数
	if m.db == nil {
		return 0, fmt.Errorf("数据库连接未初始化")
	}
	if tableName == "" || len(dataList) == 0 || len(dataList[0]) == 0 {
		return 0, fmt.Errorf("表名或数据不能为空")
	}

	// 收集并排序所有字段，确保一致性
	allFields := make(map[string]bool)
	for _, data := range dataList {
		for field := range data {
			allFields[field] = true
		}
	}

	var fields []string
	for field := range allFields {
		fields = append(fields, field)
	}
	sort.Strings(fields)
	columnsStr := strings.Join(fields, ", ")

	// 开始事务
	tx, err := m.db.Begin()
	if err != nil {
		return 0, fmt.Errorf("开始事务失败: %v", err)
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	// 分批处理
	const batchSize = 100
	var totalAffected int64

	for i := 0; i < len(dataList); i += batchSize {
		// 确定当前批次的数据范围
		end := i + batchSize
		if end > len(dataList) {
			end = len(dataList)
		}
		batch := dataList[i:end]

		// 构建批量插入的VALUES部分
		var valueStrings []string
		var valueArgs []interface{}

		for _, data := range batch {
			valuePlaceholders := make([]string, len(fields))
			for j, field := range fields {
				if val, ok := data[field]; ok {
					valuePlaceholders[j] = "?"
					valueArgs = append(valueArgs, val)
				} else {
					valuePlaceholders[j] = "NULL"
				}
			}
			valueStrings = append(valueStrings, "("+strings.Join(valuePlaceholders, ", ")+")")
		}

		// 执行批量插入
		query := fmt.Sprintf("INSERT INTO %s (%s) VALUES %s", tableName, columnsStr, strings.Join(valueStrings, ", "))
		result, err := tx.Exec(query, valueArgs...)
		if err != nil {
			return 0, fmt.Errorf("批量插入失败: %v", err)
		}

		affected, _ := result.RowsAffected()
		totalAffected += affected
	}

	// 提交事务
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("提交事务失败: %v", err)
	}

	return totalAffected, nil
}

// InsertOrUpdate: 如果where条件存在则更新，否则插入
//Data := map[string]interface{}{ /* ... */ }
//where := "devId = ? AND boxid = ?"
//_, err := Wsy.DB.InsertOrUpdate("your_table", Data, where, "123", "456")
func (m *WsyDB) InsertOrUpdate(tableName string, data map[string]interface{}, where string, args ...interface{}) (int64, error) {
	if m.db == nil {
		return 0, errors.New("数据库连接未初始化，请先调用Open方法")
	}
	// 先查是否存在
	query := fmt.Sprintf("SELECT COUNT(1) FROM %s WHERE %s", tableName, where)
	var count int64
	err := m.db.QueryRow(query, args...).Scan(&count)
	if err != nil {
		return 0, errors.New("查询失败: " + err.Error())
	}
	if count > 0 {
		// 存在则更新
		return m.Update(tableName, data, where, args...)
	} else {
		// 不存在则插入
		return m.Insert(tableName, data)
	}
}

// InsertOrUpdateKey: 高并发安全upsert，使用MySQL ON DUPLICATE KEY UPDATE
// 只适用于MySQL，表需有唯一索引或主键
// ignoreKeys: 不参与update的字段（如主键、唯一索引） ("del_list",Data,"id",'devid')
func (m *WsyDB) Upsert(tableName string, data map[string]interface{}, ignoreKeys ...string) (int64, error) {
	if m.db == nil {
		return 0, errors.New("数据库连接未初始化，请先调用Open方法")
	}
	if tableName == "" || len(data) == 0 {
		return 0, errors.New("表名或数据不能为空")
	}
	// 自动拆分逗号分隔的字符串为多个字段
	var ignoreList []string
	for _, k := range ignoreKeys {
		for _, field := range strings.Split(k, ",") {
			field = strings.TrimSpace(field)
			if field != "" {
				ignoreList = append(ignoreList, field)
			}
		}
	}
	ignore := map[string]struct{}{}
	for _, k := range ignoreList {
		ignore[k] = struct{}{}
	}
	var columns, placeholders, updates []string
	var values []interface{}
	for k, v := range data {
		columns = append(columns, k)
		placeholders = append(placeholders, "?")
		values = append(values, v)
		if _, skip := ignore[k]; !skip {
			updates = append(updates, fmt.Sprintf("%s=VALUES(%s)", k, k))
		}
	}
	if len(updates) == 0 {
		return 0, errors.New("没有可更新的字段")
	}
	query := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s) ON DUPLICATE KEY UPDATE %s",
		tableName,
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "),
		strings.Join(updates, ", "))
	result, err := m.db.Exec(query, values...)
	if err != nil {
		return 0, errors.New("执行upsert失败: " + err.Error())
	}
	rowsAffected, _ := result.RowsAffected()
	return rowsAffected, nil
}

// Row 单行

func (m *WsyDB) Row(tables []string, columns []string, where string, args ...interface{}) (map[string]interface{}, error) {
    table := strings.Join(tables, " ")
    sql := fmt.Sprintf("SELECT %s FROM %s", strings.Join(columns, ", "), table)
    if where != "" {
        sql += " WHERE " + where
    }
    sql += " LIMIT 1"
    return m.Sel(sql, args...)
}

// Rows 执行分页查询并返回结果集
// 参数：
//   - tables: 表名数组，支持多表联接
//   - columns: 要查询的列名数组
//   - where: WHERE条件子句（不含WHERE关键字）
//   - args: WHERE条件中的参数值
//   - orderBy: 排序子句（不含ORDER BY关键字）
//   - value: 可选参数，依次为：
//   - 页码(默认1)
//   - 每页记录数(默认20)
//   - 分组条件(默认空)
func (m *WsyDB) Rows(tables []string,columns []string,where string,args []interface{},orderBy string,value ...interface{},) (map[string]interface{}, error) {
	var sqlGroup,QuerySQL string
	var total int64
	page  := Str.ToInt(Str.IIF(m.Page  > 0 , Str.ToString(m.Page) , "1"))
	limit := Str.ToInt(Str.IIF(m.Limit > 0 , Str.ToString(m.Limit), "20"))
	if len(value) > 0 {
		page  = Str.ToInt(Str.IIF(Str.ToString(value[0]) > "0",Str.ToString(value[0]),"1"))
	}
	if len(value) > 1 {
		limit = Str.ToInt(Str.IIF(Str.ToString(value[1]) > "0" , Str.ToString(value[1]),Str.ToString(limit)))
	}
	if len(value) > 2 {
		sqlGroup = Str.IIF(Str.ToString(value[2]) != "", "GROUP BY "+Str.ToString(value[2]), "")
	}
	sqlWhere := Str.IIF(where   != "", "WHERE "+where, "")
	sqlOrder := Str.IIF(orderBy != "", "ORDER BY "+orderBy, "")
	offset := (page - 1) * limit
	sqlTable := strings.Join(tables, " ")
	sqlColumns := strings.Join(columns, ", ")
	// 查询总数 - 当使用GROUP BY时，需要特殊处理以获取正确的总记录数
    if sqlGroup != "" {
        QuerySQL = fmt.Sprintf("SELECT COUNT(1) FROM (SELECT 1 FROM %s %s %s) AS temp", sqlTable, sqlWhere, sqlGroup)
    } else {
        QuerySQL = fmt.Sprintf("SELECT COUNT(1) FROM %s %s", sqlTable, sqlWhere)
    }
    if m.Debug {
        Echo(m.ToSQL(QuerySQL, args...))
    }
    if err := m.db.QueryRow(QuerySQL, args...).Scan(&total); err != nil {
        return nil, errors.New("统计总数失败: " + err.Error())
    }
	pages := int((total + int64(limit) - 1) / int64(limit))
	more := int64(page*limit) < total
	start := int64(offset) + 1
	end := start + int64(limit) - 1
	if end > total {
		end = total
	}
	// 查询数据
	dataSQL := fmt.Sprintf("SELECT %s FROM %s %s %s %s %s", sqlColumns, sqlTable, sqlWhere, sqlGroup, sqlOrder, fmt.Sprintf("LIMIT %d OFFSET %d", limit, offset))
	items, err := m.Sels(dataSQL, args...)
	if err != nil {
		return nil, errors.New("查询数据失败: " + err.Error())
	}

	resMap := map[string]interface{}{
		"items":  items,
		"total":  total,
		"pages":  pages,
		"page":   page,
		"more":   more,
		"start":  start,
		"end":    end,
		"limit":  limit,
	}
	return resMap, nil
}



// RowsBuildMap 构建AND条件的基础方法，处理单个条件的构建逻辑
// 参数：
//   - conditions: 条件映射，格式为 map[string][]interface{}{"字段名": [值, 操作符]}
//
// 返回：
//   - where: WHERE条件字符串数组
//   - params: 参数数组
func (m *WsyDB) RowsBuildMap(conditions map[string][]interface{}) ([]string, []interface{}) {
	var where []string
	var params []interface{}
	
	ops := map[string]struct {
		sql   string
		exact bool
	}{
		"=":  {"= ?", true},
		"<>": {"<> ?", true},
		"!=": {"!= ?", true},
		">":  {"> ?", true},
		"<":  {"< ?", true},
		">=": {">= ?", true},
		"<=": {"<= ?", true},
		"%":  {"LIKE ?", false},
		"%%": {"LIKE ?", false},
		"*%": {"LIKE ?", false},
		"%*": {"LIKE ?", false},
		"==": {"IN", true}, // IN操作符
	}
	
	for field, cond := range conditions {
		if len(cond) < 2 {
			continue
		}
		value := cond[0]
		op, ok := cond[1].(string)
		if !ok {
			continue
		}
		
		// 验证值是否有效
		isValid := false
		switch v := value.(type) {
		case string:
			isValid = v != "" || v == "0"
		case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64, bool:
			isValid = true
		default:
			isValid = value != nil
		}
		if !isValid {
			continue
		}
		if op == "==" {
			v, ok := value.(string)
			if !ok || v == "" {
				continue
			}
			filtered := Str.ToStrArray(v)
			if len(filtered) == 0 {
				continue
			} else if len(filtered) == 1 {
				where = append(where, fmt.Sprintf("%s = ?", field))
				params = append(params, filtered[0])
			} else {
				placeholders := strings.Repeat("?,", len(filtered))
				placeholders = placeholders[:len(placeholders)-1]
				where = append(where, fmt.Sprintf("%s IN (%s)", field, placeholders))
				for _, s := range filtered {
					params = append(params, s)
				}
			}
			continue
		}
		if opInfo, exists := ops[op]; exists {
			if !opInfo.exact {
				strVal := fmt.Sprintf("%v", value)
				switch op {
				case "%", "%%":
					value = "%" + strVal + "%"
				case "*%":
					value = strVal + "%"
				case "%*":
					value = "%" + strVal
				}
			}
			where = append(where, fmt.Sprintf("%s %s", field, opInfo.sql))
			params = append(params, value)
		}
	}
	
	return where, params
}

func (m *WsyDB) RowsBuild(conditions map[string][]interface{}) (string, []interface{}) {
	where, params := m.RowsBuildMap(conditions)
	return strings.Join(where, " AND "), params
}

// RowsBuildAll 构建OR条件，支持多个条件映射的OR组合
// 参数：
//   - baseConditions: 基础条件映射
//   - orConditions: OR条件映射，格式与baseConditions相同
//
// 返回：
//   - whereStr: 合并后的WHERE条件字符串
//   - params: 合并后的参数数组
//
// 示例：
//   finalWhere, finalParams := Wsy.DB.RowsBuildAll(baseConditions, orConditions)
//   ors := map[string][]interface{}{
//  "t.sid": {"1,2,3", "=="},  // OR 条件，支持 IN 写法（"==" 会转成 IN）
    // 或者 "t.sid": {"1", "="},
    //      "t.code": {"abc", "="},
//   }
func (m *WsyDB) RowsBuildAll(baseConditions map[string][]interface{}, orConditions map[string][]interface{}) (string, []interface{}) {
	// 处理基础条件和OR条件
	baseClauses, baseParams := m.RowsBuildMap(baseConditions)
	orClauses, orParams := m.RowsBuildMap(orConditions)
	// 如果没有OR条件，直接返回基础条件
	if len(orClauses) == 0 {
		return m.RowsBuild(baseConditions)
	}
	orClause := strings.Join(orClauses, " OR ")
	if len(baseClauses) > 0 {
		return fmt.Sprintf("(%s) OR (%s)", strings.Join(baseClauses, " AND "), orClause), append(baseParams, orParams...)
	}
	return orClause, orParams
}

// Del 执行删除操作
// 参数：
//   - tableName: 要删除的表名
//   - where: WHERE条件子句（不含WHERE关键字）
//   - args: WHERE条件中的参数值
//
// 返回：
//   - rowsAffected: 受影响的行数
//   - error: 错误信息
func (m *WsyDB) Del(tableName string, where string, args ...interface{}) (int64, error) {
	if m.db == nil {
		return 0, errors.New("数据库连接未初始化，请先调用Open方法")
	}
	if tableName == "" {
		return 0, errors.New("表名不能为空")
	}
	if where == "" {
		return 0, errors.New("删除条件不能为空，请指定WHERE条件")
	}
	query := fmt.Sprintf("DELETE FROM %s WHERE %s", tableName, where)
	if m.Debug {
        Echo(m.ToSQL(query, args...))
    }
	result, err := m.db.Exec(query, args...)
	if err != nil {
		return 0, errors.New("执行删除失败: " + err.Error())
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, errors.New("获取受影响行数失败: " + err.Error())
	}
	return rowsAffected, nil
}
// Dels 批量删除，支持逗号分隔的多个值，使用IN查询
// 参数：
//   - tableName: 要删除的表名
//   - field: 要匹配的字段名
//   - values: 逗号分隔的值字符串，如 "1,2,3,4,5"
//   - additionalWhere: 额外的WHERE条件（可选）
//   - args: 额外条件的参数值
//
// 返回：
//   - rowsAffected: 受影响的行数
//   - error: 错误信息
//
// 示例：
//   Dels("dev_list", "id", "1,2,3,4,5")
//   Dels("dev_list", "id", "1,2,3,4,5", "uid = ?", "user123")
// 更进一步的简化版本
func (m *WsyDB) Dels(tableName string, field string, values string, additionalWhere string, args ...interface{}) (int64, error) {
	if m.db == nil || tableName == "" || field == "" || values == "" {
		return 0, errors.New("参数不完整")
	}
	valueArray := Str.ToStrArray(values)
	if Str.IsNull(valueArray) {
		return 0, errors.New("没有有效的删除值")
	}
	// 构建WHERE条件
	placeholders := strings.Repeat("?,", len(valueArray))
	placeholders = placeholders[:len(placeholders)-1]
	whereClause := fmt.Sprintf("%s IN (%s)", field, placeholders)
	if additionalWhere != "" {
		whereClause += " AND " + additionalWhere
	}
	// 构建参数并执行
	allArgs := make([]interface{}, 0, len(valueArray)+len(args))
	for _, v := range valueArray {
		allArgs = append(allArgs, v)
	}
	allArgs = append(allArgs, args...)
	query := fmt.Sprintf("DELETE FROM %s WHERE %s", tableName, whereClause)
	if m.Debug {
        Echo(m.ToSQL(query, allArgs...))
    }
	result, err := m.db.Exec(query, allArgs...)
	if err != nil {
		return 0, errors.New("执行批量删除失败: " + err.Error())
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, errors.New("获取受影响行数失败: " + err.Error())
	}
	return rowsAffected, nil
}

// StrIn 将逗号分隔的字符串转换为SQL IN语句使用的格式
// 参数:
//   - ids: 逗号分隔的ID字符串，如 "175,176,177,178"
//   - quotes: (可选)引号类型，默认为单引号(')，可选值：'（单引号）或 "（双引号）
//
// 返回:
//   - string: 格式化后的字符串，如 "'175','176','177','178'"
//
// 示例:
//
//	formatted := Lin.DB.StrIn("175,176,177,178")       // 返回 "'175','176','177','178'"
//	formatted := Lin.DB.StrIn("175,176,177", "\"")     // 返回 ""175","176","177""
//	formatted := Lin.DB.StrIn("abc,def", "'")          // 返回 "'abc','def'"
func (m *WsyDB) ToStrIn(ids string, quotes ...string) string {
	// 设置默认引号类型
	quote := "'"
	if len(quotes) > 0 && (quotes[0] == "'" || quotes[0] == "\"") {
		quote = quotes[0]
	}
	if ids == "" {
		return ""
	}
	idArray := strings.Split(ids, ",")
	var result strings.Builder
	for i, id := range idArray {
		id = strings.TrimSpace(id) // 去除可能的空格
		if id == "" {
			continue // 跳过空元素
		}
		if i > 0 {
			result.WriteString(",")
		}
		result.WriteString(quote)
		result.WriteString(id)
		result.WriteString(quote)
	}
	return result.String()
}

// GroupConcat 执行GROUP_CONCAT查询，自动处理长度限制
// 参数:
//   - tableName: 表名
//   - field: 要连接的字段名
//   - where: WHERE条件（可选）
//   - args: 查询参数
//
// 返回:
//   - string: GROUP_CONCAT的结果
//   - error: 错误信息
//
// 示例:
/*
boxIds, err := Wsy.DB.GroupConcat("dev_list", "boxId", "uid = ?", "user123")
// 多条件查询
boxIds, err := Wsy.DB.GroupConcat("dev_list", "boxId", "uid = ? AND state = ?", "user123", "1")
// 自定义分隔符
boxIds, err := Wsy.DB.GroupConcatWithOptions("dev_list", "boxId", "uid = ?", "|", "", "user123")
// 带排序
boxIds, err := Wsy.DB.GroupConcatWithOptions("dev_list", "boxId", "uid = ?", ",", "id DESC", "user123")
-- 基础查询
SELECT GROUP_CONCAT(boxId) FROM dev_list WHERE uid = ?

-- 自定义分隔符和排序
SELECT GROUP_CONCAT(boxId SEPARATOR '|' ORDER BY id DESC) FROM dev_list WHERE uid = ?
*/
func (m *WsyDB) GroupConcat(tableName, field, where string, args ...interface{}) (string, error) {
	if m.db == nil {
		return "", errors.New("数据库连接未初始化，请先调用Open方法")
	}
	// 设置GROUP_CONCAT最大长度，避免截断（10MB）
	_, err := m.Exec("SET SESSION group_concat_max_len = 10485760")
	if err != nil {
		return "", errors.New("设置GROUP_CONCAT长度失败: " + err.Error())
	}
	// 构建SQL查询
	query := fmt.Sprintf("SELECT GROUP_CONCAT(%s) FROM %s", field, tableName)
	if where != "" {
		query += " WHERE " + where
	}
	// 执行查询
	row, err := m.Sel(query, args...)
	if err != nil {
		return "", err
	}
	if len(row) == 0 {
		return "", nil
	}
	// 获取第一个字段的值（GROUP_CONCAT结果）
	for _, value := range row {
		if value == nil {
			return "", nil
		}
		if str, ok := value.(string); ok {
			return str, nil
		}
		return fmt.Sprintf("%v", value), nil
	}
	return "", nil
}

// GroupConcatWithOptions 执行GROUP_CONCAT查询，支持更多选项
// 参数:
//   - tableName: 表名
//   - field: 要连接的字段名
//   - where: WHERE条件（可选）
//   - separator: 分隔符（可选，默认为逗号）
//   - orderBy: 排序字段（可选）
//   - args: 查询参数
//
// 返回:
//   - string: GROUP_CONCAT的结果
//   - error: 错误信息
//
// 示例:
//   result, err := Wsy.DB.GroupConcatWithOptions("dev_list", "boxId", "uid = ?", "|", "id DESC", "user123")
func (m *WsyDB) GroupConcatWithOptions(tableName, field, where, separator, orderBy string, args ...interface{}) (string, error) {
	if m.db == nil {
		return "", errors.New("数据库连接未初始化，请先调用Open方法")
	}
	
	// 设置GROUP_CONCAT最大长度，避免截断（10MB）
	_, err := m.Exec("SET SESSION group_concat_max_len = 10485760")
	if err != nil {
		return "", errors.New("设置GROUP_CONCAT长度失败: " + err.Error())
	}
	
	// 设置默认分隔符
	if separator == "" {
		separator = ","
	}
	
	// 构建SQL查询
	query := fmt.Sprintf("SELECT GROUP_CONCAT(%s SEPARATOR '%s')", field, separator)
	if orderBy != "" {
		query += fmt.Sprintf(" ORDER BY %s", orderBy)
	}
	query += fmt.Sprintf(" FROM %s", tableName)
	if where != "" {
		query += " WHERE " + where
	}
	
	// 执行查询
	row, err := m.Sel(query, args...)
	if err != nil {
		return "", err
	}
	
	if len(row) == 0 {
		return "", nil
	}
	
	// 获取第一个字段的值（GROUP_CONCAT结果）
	for _, value := range row {
		if value == nil {
			return "", nil
		}
		if str, ok := value.(string); ok {
			return str, nil
		}
		return fmt.Sprintf("%v", value), nil
	}
	
	return "", nil
}

// Empty 检查查询结果是否为空：优先返回 err，否则 empty 为 true 时返回 tip 错误
func (m *WsyDB) Empty(empty bool, err error, tip string) error {
	if err != nil {
		return err
	}
	if empty {
		return errors.New(tip)
	}
	return nil
}
