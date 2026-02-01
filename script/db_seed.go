package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	_ "github.com/denisenkom/go-mssqldb"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	"gopkg.in/yaml.v3"
	_ "modernc.org/sqlite"
)

type Config struct {
	Mysql struct {
		Host     string `yaml:"Host"`
		Port     int    `yaml:"Port"`
		User     string `yaml:"User"`
		Password string `yaml:"Password"`
		Database string `yaml:"Database"`
	} `yaml:"Mysql"`
}

type Column struct {
	Name, DataType, ColumnType                                            string
	DefaultValue                                                          string
	IsNullable, HasDefault, IsAuto, IsGenerated, IsPrimaryKey, IsUnsigned bool
	CharMaxLen, NumPrecision, NumScale                                    int64
	IsBool, IsText, IsNumeric, IsDate, IsTime, IsJSON, IsUUID             bool
}

type DBType string

const (
	DBMySQL  DBType = "mysql"
	DBPG     DBType = "postgres"
	DBSQLite DBType = "sqlite"
	DBSQLSrv DBType = "sqlserver"
)

type Progress struct {
	TotalTables, CompletedTables, TotalTargetRows, TotalInsertedRows, TotalFailedBatches int64
	Start                                                                                time.Time
	MaxHeap, MaxSys                                                                      uint64
}

func main() {
	var (
		dbType      = flag.String("db", "mysql", "mysql|postgres|sqlite|sqlserver")
		dsn         = flag.String("dsn", "", "dsn (optional)")
		host        = flag.String("host", "", "host")
		port        = flag.Int("port", 0, "port")
		user        = flag.String("user", "", "user")
		pass        = flag.String("password", "", "password")
		database    = flag.String("database", "", "database")
		schema      = flag.String("schema", "", "schema (pg)")
		perTable    = flag.Int("per-table", 1000000, "rows per table")
		batchSize   = flag.Int("batch-size", 1000, "batch size")
		concurrency = flag.Int("concurrency", 1, "table concurrency")
		seed        = flag.Int64("seed", time.Now().UnixNano(), "random seed")
	)
	flag.Parse()

	dt := DBType(strings.ToLower(*dbType))
	dsnFinal, driver := buildDSN(dt, *dsn, *host, *port, *user, *pass, *database, *schema)
	log.Printf("db=%s driver=%s dsn=%s", dt, driver, maskDSN(dsnFinal))

	db, err := sql.Open(driver, dsnFinal)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		log.Fatal(err)
	}

	tables, err := listTables(db, dt, *database, *schema)
	if err != nil {
		log.Fatal(err)
	}

	// 筛选出属于审核模块的表
	auditTables := []string{}
	for _, t := range tables {
		if strings.HasPrefix(t, "audit_") || strings.HasPrefix(t, "rule_") {
			auditTables = append(auditTables, t)
		}
	}

	if len(auditTables) == 0 {
		log.Println("没有找到审核模块相关的表 (audit_ 或 rule_ 开头)")
		return
	}

	tables = auditTables
	log.Printf("发现审核模块表: %v", tables)

	progress := &Progress{TotalTables: int64(len(tables)), TotalTargetRows: int64(len(tables) * (*perTable)), Start: time.Now()}
	stop := make(chan struct{})
	go report(progress, stop)

	// 创建一个全局任务队列
	type task struct {
		table string
		seed  int64
	}
	taskChan := make(chan task, *concurrency*2)

	// 启动工作池
	var wg sync.WaitGroup
	for i := 0; i < *concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			// 每个工作协程有自己的随机数生成器，避免锁竞争
			rng := rand.New(rand.NewSource(*seed + int64(workerID)))
			for t := range taskChan {
				// 打印正在处理的表
				if progress.TotalInsertedRows%10000 == 0 {
					log.Printf("Worker %d processing table %s", workerID, t.table)
				}
				cols, err := listColumns(db, dt, t.table, *database, *schema)
				if err != nil {
					log.Printf("worker %d list columns error: %v", workerID, err)
					continue
				}
				insertCols := make([]Column, 0, len(cols))
				for _, c := range cols {
					if c.IsAuto || c.IsGenerated || c.HasDefault {
						continue
					}
					insertCols = append(insertCols, c)
				}

				var insertErr error
				if len(insertCols) == 0 {
					insertErr = insertDefaultRows(db, dt, t.table, *batchSize)
				} else {
					insertErr = insertBatch(db, dt, t.table, insertCols, *batchSize, rng)
				}

				if insertErr != nil {
					atomic.AddInt64(&progress.TotalFailedBatches, 1)
					// 只在第一次失败时打印错误详情，避免刷屏
					if progress.TotalFailedBatches <= 10 {
						log.Printf("Table %s batch insert failed: %v", t.table, insertErr)
					}
				} else {
					atomic.AddInt64(&progress.TotalInsertedRows, int64(*batchSize))
				}
			}
		}(i)
	}

	// 派发任务：持续将 100 万条的任务拆解成批次推入队列
	go func() {
		for {
			allDone := true
			for _, t := range tables {
				// 检查该表是否已达到目标行数
				// 注意：这里需要一个更精确的按表计数，但为了全局并发，我们先采用简单循环
				// 实际上我们可以计算总批次
				_ = t
			}

			// 修正派发逻辑：按总批次派发，不分表
			totalBatches := int(progress.TotalTargetRows / int64(*batchSize))
			for i := 0; i < totalBatches; i++ {
				// 轮询表名
				tableName := tables[i%len(tables)]
				taskChan <- task{table: tableName, seed: *seed + int64(i)}
			}
			close(taskChan)
			allDone = true
			if allDone {
				break
			}
		}
	}()

	wg.Wait()
	close(stop)

	elapsed := time.Since(progress.Start)
	log.Printf("done tables=%d completed=%d inserted=%d/%d failedBatches=%d elapsed=%s maxHeap=%s maxSys=%s",
		progress.TotalTables, progress.CompletedTables, progress.TotalInsertedRows, progress.TotalTargetRows, progress.TotalFailedBatches,
		elapsed, formatBytes(progress.MaxHeap), formatBytes(progress.MaxSys))
}

func report(p *Progress, stop <-chan struct{}) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			var m runtime.MemStats
			runtime.ReadMemStats(&m)
			if m.Alloc > p.MaxHeap {
				p.MaxHeap = m.Alloc
			}
			if m.Sys > p.MaxSys {
				p.MaxSys = m.Sys
			}
			elapsed := time.Since(p.Start)
			log.Printf("progress tables=%d/%d inserted=%d/%d failedBatches=%d elapsed=%s heap=%s sys=%s goroutines=%d",
				p.CompletedTables, p.TotalTables, p.TotalInsertedRows, p.TotalTargetRows, p.TotalFailedBatches,
				elapsed, formatBytes(m.Alloc), formatBytes(m.Sys), runtime.NumGoroutine())
		case <-stop:
			return
		}
	}
}

func buildDSN(dt DBType, dsn, host string, port int, user, pass, database, schema string) (string, string) {
	if dsn != "" {
		return dsn, driverName(dt)
	}
	if dt == DBMySQL {
		cfg, err := loadConfig("d:/gowork/piaowu/conf/config.yaml")
		if err == nil && host == "" {
			host, port, user, pass, database = cfg.Mysql.Host, cfg.Mysql.Port, cfg.Mysql.User, cfg.Mysql.Password, cfg.Mysql.Database
		}
		if port == 0 {
			port = 3306
		}
		return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true&charset=utf8mb4&loc=Local", user, pass, host, port, database), "mysql"
	}
	if dt == DBPG {
		if port == 0 {
			port = 5432
		}
		if schema == "" {
			schema = "public"
		}
		return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable search_path=%s", host, port, user, pass, database, schema), "postgres"
	}
	if dt == DBSQLite {
		if dsn == "" {
			dsn = "./data.db"
		}
		return dsn, "sqlite"
	}
	if dt == DBSQLSrv {
		if port == 0 {
			port = 1433
		}
		return fmt.Sprintf("sqlserver://%s:%s@%s:%d?database=%s", user, pass, host, port, database), "sqlserver"
	}
	return dsn, driverName(dt)
}

func driverName(dt DBType) string {
	switch dt {
	case DBMySQL:
		return "mysql"
	case DBPG:
		return "postgres"
	case DBSQLite:
		return "sqlite"
	case DBSQLSrv:
		return "sqlserver"
	default:
		return string(dt)
	}
}

func loadConfig(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func listTables(db *sql.DB, dt DBType, database, schema string) ([]string, error) {
	switch dt {
	case DBMySQL:
		rows, err := db.Query(`SELECT table_name FROM information_schema.tables WHERE table_schema = ? AND table_type='BASE TABLE'`, database)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		return scanStrings(rows)
	case DBPG:
		if schema == "" {
			schema = "public"
		}
		rows, err := db.Query(`SELECT table_name FROM information_schema.tables WHERE table_schema = $1 AND table_type='BASE TABLE'`, schema)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		return scanStrings(rows)
	case DBSQLite:
		rows, err := db.Query(`SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'`)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		return scanStrings(rows)
	case DBSQLSrv:
		rows, err := db.Query(`SELECT TABLE_NAME FROM INFORMATION_SCHEMA.TABLES WHERE TABLE_TYPE='BASE TABLE'`)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		return scanStrings(rows)
	default:
		return nil, fmt.Errorf("unsupported db type")
	}
}

func scanStrings(rows *sql.Rows) ([]string, error) {
	var res []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, err
		}
		res = append(res, s)
	}
	return res, nil
}

func listColumns(db *sql.DB, dt DBType, table, database, schema string) ([]Column, error) {
	switch dt {
	case DBMySQL:
		rows, err := db.Query(`SELECT column_name, data_type, is_nullable, column_default, character_maximum_length, numeric_precision, numeric_scale, column_type, column_key, extra FROM information_schema.columns WHERE table_schema = ? AND table_name = ? ORDER BY ordinal_position`, database, table)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		var cols []Column
		for rows.Next() {
			var name, dataType, isNullable, columnDefault, columnType, columnKey, extra sql.NullString
			var charMax, numPrec, numScale sql.NullInt64
			if err := rows.Scan(&name, &dataType, &isNullable, &columnDefault, &charMax, &numPrec, &numScale, &columnType, &columnKey, &extra); err != nil {
				return nil, err
			}
			col := Column{Name: name.String, DataType: strings.ToLower(dataType.String), ColumnType: columnType.String,
				IsNullable: isNullable.String == "YES", HasDefault: columnDefault.Valid, CharMaxLen: charMax.Int64,
				NumPrecision: numPrec.Int64, NumScale: numScale.Int64, IsPrimaryKey: columnKey.String == "PRI",
				IsAuto: strings.Contains(strings.ToLower(extra.String), "auto_increment"), IsGenerated: strings.Contains(strings.ToLower(extra.String), "generated")}
			cols = append(cols, normalizeColumn(col))
		}
		return cols, nil
	case DBPG:
		if schema == "" {
			schema = "public"
		}
		rows, err := db.Query(`SELECT column_name, data_type, is_nullable, column_default, character_maximum_length, numeric_precision, numeric_scale FROM information_schema.columns WHERE table_schema = $1 AND table_name = $2 ORDER BY ordinal_position`, schema, table)

		if err != nil {
			return nil, err
		}
		defer rows.Close()
		var cols []Column
		for rows.Next() {
			var name, dataType, isNullable, columnDefault sql.NullString
			var charMax, numPrec, numScale sql.NullInt64
			if err := rows.Scan(&name, &dataType, &isNullable, &columnDefault, &charMax, &numPrec, &numScale); err != nil {
				return nil, err
			}
			col := Column{Name: name.String, DataType: strings.ToLower(dataType.String), IsNullable: isNullable.String == "YES",
				HasDefault: columnDefault.Valid, CharMaxLen: charMax.Int64, NumPrecision: numPrec.Int64, NumScale: numScale.Int64,
				IsAuto: strings.Contains(columnDefault.String, "nextval(")}
			cols = append(cols, normalizeColumn(col))
		}
		return cols, nil
	case DBSQLite:
		rows, err := db.Query(fmt.Sprintf(`PRAGMA table_info(%s)`, quoteIdent(dt, table)))
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		var cols []Column
		for rows.Next() {
			var cid int
			var name, dataType string
			var notnull int
			var dflt sql.NullString
			var pk int
			if err := rows.Scan(&cid, &name, &dataType, &notnull, &dflt, &pk); err != nil {
				return nil, err
			}
			col := Column{Name: name, DataType: strings.ToLower(dataType), IsNullable: notnull == 0, HasDefault: dflt.Valid,
				DefaultValue: dflt.String, IsPrimaryKey: pk == 1, IsAuto: pk == 1}
			cols = append(cols, normalizeColumn(col))
		}
		return cols, nil
	case DBSQLSrv:
		rows, err := db.Query(`SELECT COLUMN_NAME, DATA_TYPE, IS_NULLABLE, COLUMN_DEFAULT, CHARACTER_MAXIMUM_LENGTH, NUMERIC_PRECISION, NUMERIC_SCALE FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_NAME = ? ORDER BY ORDINAL_POSITION`, table)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		var cols []Column
		for rows.Next() {
			var name, dataType, isNullable, columnDefault sql.NullString
			var charMax, numPrec, numScale sql.NullInt64
			if err := rows.Scan(&name, &dataType, &isNullable, &columnDefault, &charMax, &numPrec, &numScale); err != nil {
				return nil, err
			}
			col := Column{Name: name.String, DataType: strings.ToLower(dataType.String), IsNullable: isNullable.String == "YES",
				HasDefault: columnDefault.Valid, CharMaxLen: charMax.Int64, NumPrecision: numPrec.Int64, NumScale: numScale.Int64}
			cols = append(cols, normalizeColumn(col))
		}
		return cols, nil
	default:
		return nil, fmt.Errorf("unsupported db type")
	}
}

func normalizeColumn(c Column) Column {
	dt := c.DataType
	c.IsBool = strings.Contains(dt, "bool") || dt == "tinyint(1)" || dt == "bit"
	c.IsJSON = strings.Contains(dt, "json")
	c.IsUUID = strings.Contains(dt, "uuid") || strings.Contains(strings.ToLower(c.Name), "uuid")
	c.IsTime = strings.Contains(dt, "time")
	c.IsDate = strings.Contains(dt, "date")
	c.IsNumeric = strings.Contains(dt, "int") || strings.Contains(dt, "decimal") || strings.Contains(dt, "numeric") || strings.Contains(dt, "double") || strings.Contains(dt, "float")
	c.IsText = strings.Contains(dt, "char") || strings.Contains(dt, "text") || strings.Contains(dt, "clob")
	if strings.Contains(strings.ToLower(c.ColumnType), "unsigned") {
		c.IsUnsigned = true
	}
	return c
}

func seedTable(db *sql.DB, dt DBType, table, database, schema string, perTable, batchSize int, seed int64, p *Progress) error {
	cols, err := listColumns(db, dt, table, database, schema)
	if err != nil {
		return err
	}
	insertCols := make([]Column, 0, len(cols))
	for _, c := range cols {
		if c.IsAuto || c.IsGenerated || c.HasDefault {
			continue
		}
		insertCols = append(insertCols, c)
	}

	rng := rand.New(rand.NewSource(seed + int64(len(table))))
	inserted := 0
	for inserted < perTable {
		n := batchSize
		if inserted+n > perTable {
			n = perTable - inserted
		}
		var err error
		if len(insertCols) == 0 {
			err = insertDefaultRows(db, dt, table, n)
		} else {
			err = insertBatch(db, dt, table, insertCols, n, rng)
		}
		if err != nil {
			atomic.AddInt64(&p.TotalFailedBatches, 1)
			if n > 1 {
				_ = insertBatch(db, dt, table, insertCols, max(1, n/2), rng)
			}
		}
		inserted += n
		atomic.AddInt64(&p.TotalInsertedRows, int64(n))
	}
	return nil
}

func insertDefaultRows(db *sql.DB, dt DBType, table string, count int) error {
	q := ""
	switch dt {
	case DBMySQL:
		q = fmt.Sprintf("INSERT INTO %s () VALUES ()", quoteIdent(dt, table))
	default:
		q = fmt.Sprintf("INSERT INTO %s DEFAULT VALUES", quoteIdent(dt, table))
	}
	for i := 0; i < count; i++ {
		if _, err := db.Exec(q); err != nil {
			return err
		}
	}
	return nil
}

func insertBatch(db *sql.DB, dt DBType, table string, cols []Column, rows int, rng *rand.Rand) error {
	query, args := buildInsert(dt, table, cols, rows, rng)
	_, err := db.Exec(query, args...)
	return err
}

func buildInsert(dt DBType, table string, cols []Column, rows int, rng *rand.Rand) (string, []any) {
	colNames := make([]string, 0, len(cols))
	for _, c := range cols {
		colNames = append(colNames, quoteIdent(dt, c.Name))
	}
	args := make([]any, 0, rows*len(cols))
	values := make([]string, 0, rows)
	idx := 1
	for i := 0; i < rows; i++ {
		placeholders := make([]string, 0, len(cols))
		for _, c := range cols {
			placeholders = append(placeholders, placeholder(dt, idx))
			args = append(args, genValue(c, rng))
			idx++
		}
		values = append(values, "("+strings.Join(placeholders, ",")+")")
	}
	q := fmt.Sprintf("INSERT INTO %s (%s) VALUES %s", quoteIdent(dt, table), strings.Join(colNames, ","), strings.Join(values, ","))
	return q, args
}

func placeholder(dt DBType, idx int) string {
	switch dt {
	case DBPG:
		return fmt.Sprintf("$%d", idx)
	case DBSQLSrv:
		return fmt.Sprintf("@p%d", idx)
	default:
		return "?"
	}
}

func quoteIdent(dt DBType, s string) string {
	switch dt {
	case DBMySQL:
		return "`" + s + "`"
	case DBSQLSrv:
		return "[" + s + "]"
	default:
		return "\"" + s + "\""
	}
}

func genValue(c Column, rng *rand.Rand) any {
	if c.IsNullable && rng.Intn(20) == 0 {
		return nil
	}
	if c.IsBool {
		return rng.Intn(2) == 1
	}
	if c.IsNumeric {
		if strings.Contains(c.DataType, "decimal") || strings.Contains(c.DataType, "numeric") || c.NumScale > 0 {
			return float64(rng.Intn(1000000)) / 100.0
		}
		if strings.Contains(c.DataType, "tinyint") {
			return rng.Intn(127)
		}
		if strings.Contains(c.DataType, "smallint") {
			return rng.Intn(32767)
		}
		return rng.Intn(1000000)
	}
	if c.IsDate || c.IsTime {
		return time.Now().Add(time.Duration(rng.Intn(86400)) * time.Second).Format("2006-01-02 15:04:05")
	}
	if c.IsUUID {
		return randUUID(rng)
	}
	if c.IsJSON {
		m := map[string]any{"id": rng.Intn(1000000), "ok": rng.Intn(2) == 1}
		b, _ := json.Marshal(m)
		return string(b)
	}
	maxLen := int(c.CharMaxLen)
	if maxLen <= 0 || maxLen > 128 {
		maxLen = 64
	}
	if c.IsText {
		return randString(rng, maxLen)
	}
	return randString(rng, 32)
}

func randString(rng *rand.Rand, n int) string {
	letters := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rng.Intn(len(letters))]
	}
	return string(b)
}

func randUUID(rng *rand.Rand) string {
	b := make([]byte, 16)
	for i := range b {
		b[i] = byte(rng.Intn(256))
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%02x%02x%02x%02x-%02x%02x-%02x%02x-%02x%02x-%02x%02x%02x%02x%02x%02x",
		b[0], b[1], b[2], b[3], b[4], b[5], b[6], b[7], b[8], b[9], b[10], b[11], b[12], b[13], b[14], b[15])
}

func maskDSN(dsn string) string {
	if i := strings.Index(dsn, ":"); i >= 0 {
		if j := strings.Index(dsn, "@"); j > i {
			return dsn[:i+1] + "***" + dsn[j:]
		}
	}
	return dsn
}

func formatBytes(b uint64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%dB", b)
	}
	div, exp := uint64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%ciB", float64(b)/float64(div), "KMGTPE"[exp])
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
