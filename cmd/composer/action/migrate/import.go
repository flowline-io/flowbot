package migrate

import (
	"database/sql"
	"fmt"
	storeMigrate "github.com/flowline-io/flowbot/internal/store/migrate"
	_ "github.com/go-sql-driver/mysql"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/mysql"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/urfave/cli/v2"
	"gopkg.in/yaml.v3"
	"io"
	"os"
)

func ImportAction(c *cli.Context) error {
	conffile := c.String("config")

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

	db, _ := sql.Open("mysql", dsn)
	driver, _ := mysql.WithInstance(db, &mysql.Config{})

	d, err := iofs.New(storeMigrate.Fs, "migrations")
	if err != nil {
		panic(err)
	}
	m, err := migrate.NewWithInstance("iofs", d, "mysql", driver)
	if err != nil {
		panic(err)
	}
	err = m.Up()
	if err != nil {
		panic(err)
	}
	fmt.Println("done")
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
