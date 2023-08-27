package generator

import (
	_ "embed"
	"errors"
	"fmt"
	"github.com/sysatom/flowbot/pkg/utils"
	"github.com/urfave/cli/v2"
	"os"
)

//go:embed tmpl/vendor.tmpl
var vendorTemple string

const VendorsPath = "./server/extra/vendors"

func VendorAction(c *cli.Context) error {
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

	fmt.Println("done")
	return nil
}
