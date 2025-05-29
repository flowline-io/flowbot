package doc

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/flowline-io/flowbot/pkg/flog"
	_ "github.com/go-sql-driver/mysql" //revive:disable
	"github.com/goccy/go-yaml"
	"github.com/jmoiron/sqlx"
	"github.com/urfave/cli/v3"
)

type Column struct {
	TableCatalog           string         `db:"TABLE_CATALOG"`
	TableSchema            string         `db:"TABLE_SCHEMA"`
	TableName              string         `db:"TABLE_NAME"`
	ColumnName             string         `db:"COLUMN_NAME"`
	OrdinalPosition        int            `db:"ORDINAL_POSITION"`
	ColumnDefault          sql.NullString `db:"COLUMN_DEFAULT"`
	IsNullable             string         `db:"IS_NULLABLE"`
	DataType               string         `db:"DATA_TYPE"`
	CharacterMaximumLength sql.NullInt64  `db:"CHARACTER_MAXIMUM_LENGTH"`
	CharacterOctetLength   sql.NullInt64  `db:"CHARACTER_OCTET_LENGTH"`
	NumericPrecision       sql.NullInt64  `db:"NUMERIC_PRECISION"`
	NumericScale           sql.NullInt64  `db:"NUMERIC_SCALE"`
	DatetimePrecision      sql.NullInt64  `db:"DATETIME_PRECISION"`
	CharacterSetName       sql.NullString `db:"CHARACTER_SET_NAME"`
	CollationName          sql.NullString `db:"COLLATION_NAME"`
	ColumnType             string         `db:"COLUMN_TYPE"`
	ColumnKey              string         `db:"COLUMN_KEY"`
	Extra                  string         `db:"EXTRA"`
	Privileges             string         `db:"PRIVILEGES"`
	ColumnComment          string         `db:"COLUMN_COMMENT"`
	GenerationExpression   string         `db:"GENERATION_EXPRESSION"`
	SrsID                  sql.NullInt64  `db:"SRS_ID"`
}

type Table struct {
	TableCatalog   string         `db:"TABLE_CATALOG"`
	TableSchema    string         `db:"TABLE_SCHEMA"`
	TableName      string         `db:"TABLE_NAME"`
	TableType      string         `db:"TABLE_TYPE"`
	Engine         string         `db:"ENGINE"`
	Version        int            `db:"VERSION"`
	RowFormat      string         `db:"ROW_FORMAT"`
	TableRows      int            `db:"TABLE_ROWS"`
	AvgRowLength   int            `db:"AVG_ROW_LENGTH"`
	DataLength     int            `db:"DATA_LENGTH"`
	MaxDataLength  int            `db:"MAX_DATA_LENGTH"`
	IndexLength    int            `db:"INDEX_LENGTH"`
	DataFree       int            `db:"DATA_FREE"`
	AutoIncrement  sql.NullInt64  `db:"AUTO_INCREMENT"`
	CreateTime     string         `db:"CREATE_TIME"`
	UpdateTime     sql.NullString `db:"UPDATE_TIME"`
	CheckTime      sql.NullString `db:"CHECK_TIME"`
	TableCollation string         `db:"TABLE_COLLATION"`
	Checksum       sql.NullInt64  `db:"CHECKSUM"`
	CreateOptions  string         `db:"CREATE_OPTIONS"`
	TableComment   string         `db:"TABLE_COMMENT"`
}

func NullString2String(s sql.NullString) string {
	if s.Valid {
		return s.String
	} else {
		return "NULL"
	}
}

func SchemaAction(ctx context.Context, c *cli.Command) error {
	conffile := c.String("config")
	database := c.String("database")

	file, err := os.Open(filepath.Clean(conffile))
	if err != nil {
		flog.Panic(err.Error())
	}

	config := configType{}

	data, err := io.ReadAll(file)
	if err != nil {
		flog.Panic(err.Error())
	}
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		flog.Panic(err.Error())
	}

	if config.StoreConfig.UseAdapter != "mysql" {
		flog.Panic("error adapter")
	}
	if config.StoreConfig.Adapters.Mysql.DSN == "" {
		flog.Panic("error adapter dsn")
	}
	dsn := config.StoreConfig.Adapters.Mysql.DSN

	// Conn
	db, err := sqlx.Open("mysql", dsn)
	if err != nil {
		flog.Panic(err.Error())
	}
	defer func() {
		_ = db.Close()
	}()

	// Tables
	var tables []Table
	err = db.Select(&tables, "SELECT * FROM INFORMATION_SCHEMA.TABLES WHERE TABLE_SCHEMA = ?", database)
	if err != nil {
		flog.Panic(err.Error())
	}

	// Markdown
	var markdown strings.Builder
	for _, table := range tables {
		var columns []Column
		err = db.Select(&columns, "SELECT * FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?", database, table.TableName)
		if err != nil {
			flog.Panic(err.Error())
		}

		var comment strings.Builder
		if utf8.RuneCountInString(table.TableComment) > 0 {
			comment.WriteString(fmt.Sprintf(" ( %s ) ", table.TableComment))
		}
		markdown.WriteString(fmt.Sprintf("## %s %s\n\n", table.TableName, comment.String()))
		markdown.WriteString("| COLUMN_NAME |    COLUMN_TYPE   | COLUMN_DEFAULT | IS_NULLABLE | COLUMN_KEY |     EXTRA      | COLUMN_COMMENT |\n")
		markdown.WriteString("|-------------|------------------|----------------|-------------|------------|----------------|----------------|\n")

		for _, column := range columns {
			markdown.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %s | %s | %s |\n", column.ColumnName, column.ColumnType, NullString2String(column.ColumnDefault), column.IsNullable, column.ColumnKey, column.Extra, column.ColumnComment))
		}

		markdown.WriteString("\n\n")
	}

	// Write File
	err = os.WriteFile("./docs/schema.md", []byte(markdown.String()), 0644)
	if err != nil {
		flog.Panic(err.Error())
	}

	fmt.Println("See schema.md")
	return nil
}

type configType struct {
	StoreConfig struct {
		UseAdapter string `json:"use_adapter" yaml:"use_adapter"`
		Adapters   struct {
			Mysql struct {
				DSN string `json:"dsn" yaml:"dsn"`
			} `json:"mysql" yaml:"mysql"`
		} `json:"adapters" yaml:"adapters"`
	} `json:"store_config" yaml:"store_config"`
}
