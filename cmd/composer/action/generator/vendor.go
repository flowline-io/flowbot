package generator

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"os"

	"github.com/flowline-io/flowbot/pkg/utils"
	"github.com/urfave/cli/v3"
)

//go:embed tmpl/vendor.tmpl
var vendorTemple string

const VendorsPath = "./pkg/providers"

func VendorAction(ctx context.Context, c *cli.Command) error {
	vendor := c.String("name")
	if vendor == "" {
		return errors.New("vendor name args error")
	}

	_, err := os.Stat(VendorsPath)
	if os.IsNotExist(err) {
		return errors.New("vendors NotExist")
	}
	dir := fmt.Sprintf("%s/%s", VendorsPath, vendor)
	_, err = os.Stat(dir)
	if !os.IsNotExist(err) {
		return fmt.Errorf("vendor %s exist", vendor)
	}
	err = os.Mkdir(dir, os.ModePerm)
	if err != nil {
		return err
	}

	data := struct {
		VendorName string
		ClassName  string
	}{
		VendorName: vendor,
		ClassName:  utils.FirstUpper(vendor),
	}

	err = os.WriteFile(fmt.Sprintf("%s/%s/%s.go", VendorsPath, vendor, vendor), parseTemplate(vendorTemple, data), os.ModePerm)
	if err != nil {
		return err
	}

	_, _ = fmt.Println("done")
	return nil
}
