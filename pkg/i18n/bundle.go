package i18n

import (
	"embed"
	"io/fs"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/pelletier/go-toml/v2"
	"golang.org/x/text/language"
)

//go:embed locales/*.toml
var localeFS embed.FS

var defaultBundle *i18n.Bundle

func init() {
	defaultBundle = i18n.NewBundle(language.English)
	defaultBundle.RegisterUnmarshalFunc("toml", toml.Unmarshal)
	if err := fs.WalkDir(localeFS, "locales", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		data, err := localeFS.ReadFile(path)
		if err != nil {
			return err
		}
		_, err = defaultBundle.ParseMessageFileBytes(data, path)
		return err
	}); err != nil {
		panic("i18n: load locales: " + err.Error())
	}
}

// Bundle loads and returns the shared message bundle.
func Bundle() *i18n.Bundle {
	return defaultBundle
}
