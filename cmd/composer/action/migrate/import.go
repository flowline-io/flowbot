package migrate

import (
	"database/sql"
	"encoding/json"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/mysql"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	extraMigrate "github.com/sysatom/flowbot/internal/store/migrate"
	"github.com/tinode/jsonco"
	"github.com/urfave/cli/v2"
	"os"
)

func ImportAction(c *cli.Context) error {
	conffile := c.String("config")

	file, err := os.Open(conffile)
	if err != nil {
		panic(err)
	}

	config := configType{}
	jr := jsonco.New(file)
	if err = json.NewDecoder(jr).Decode(&config); err != nil {
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

	d, err := iofs.New(extraMigrate.Fs, "migrations")
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
		UseAdapter string `json:"use_adapter"`
		Adapters   struct {
			Mysql struct {
				DSN string `json:"dsn"`
			} `json:"mysql"`
		} `json:"adapters"`
	} `json:"store_config"`
}
