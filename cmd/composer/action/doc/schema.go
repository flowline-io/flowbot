package doc

import (
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/urfave/cli/v2"
	"gopkg.in/yaml.v3"
	"io"
	"os"
	"strings"
	"unicode/utf8"
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

func SchemaAction(c *cli.Context) error {
	conffile := c.String("config")
	database := c.String("database")

	file, err := os.Open(conffile)
	if err != nil {
		panic(err)
	}

	config := configType{}

	data, err := io.ReadAll(file)
	if err != nil {
		panic(err)
	}
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		panic(err)
	}

	if config.StoreConfig.UseAdapter != "mysql" {
		panic("error adapter")
	}
	if config.StoreConfig.Adapters.Mysql.DSN == "" {
		panic("error adapter dsn")
	}
	dsn := config.StoreConfig.Adapters.Mysql.DSN

	// Conn
	db, err := sqlx.Open("mysql", dsn)
	if err != nil {
		fmt.Println(err)
		panic(err)
	}
	defer db.Close()

	// Tables
	var tables []Table
	err = db.Select(&tables, "SELECT * FROM INFORMATION_SCHEMA.TABLES WHERE TABLE_SCHEMA = ?", database)
	if err != nil {
		fmt.Println(err)
		panic(err)
	}

	// Markdown
	var markdown strings.Builder
	for _, table := range tables {
		var columns []Column
		err = db.Select(&columns, "SELECT * FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?", database, table.TableName)
		if err != nil {
			fmt.Println(err)
			panic(err)
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
		fmt.Println(err)
		panic(err)
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
