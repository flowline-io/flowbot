package migrate

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"

	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/urfave/cli/v3"
)

const path = "./internal/store/migrate/migrations"

func MigrationAction(ctx context.Context, c *cli.Command) error {
	name := c.String("name")
	if name == "" {
		return errors.New("error name")
	}

	// find current version
	entry, err := os.ReadDir(path)
	if err != nil {
		flog.Panic(err.Error())
	}
	maxNo := 0
	for _, item := range entry {
		info, err := item.Info()
		if err != nil {
			log.Println(err)
			continue
		}
		if info.IsDir() {
			continue
		}
		reg, err := regexp.Compile(`\d{6}`)
		if err != nil {
			flog.Panic(err.Error())
		}
		str := reg.FindString(info.Name())

		no, _ := strconv.Atoi(str)
		if no > maxNo {
			maxNo = no
		}
	}

	maxNo++

	upName := fmt.Sprintf("%06d_%s.up.sql", maxNo, name)
	downName := fmt.Sprintf("%06d_%s.down.sql", maxNo, name)

	_, err = os.Create(fmt.Sprintf("%s/%s", path, upName))
	if err != nil {
		flog.Panic(err.Error())
	}
	_, _ = fmt.Printf("Created %s\n", upName)
	_, err = os.Create(fmt.Sprintf("%s/%s", path, downName))
	if err != nil {
		flog.Panic(err.Error())
	}
	_, _ = fmt.Printf("Created %s\n", downName)
	_, _ = fmt.Println("done")
	return nil
}
