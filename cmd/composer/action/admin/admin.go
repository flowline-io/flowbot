// Package admin provides management commands for Flowbot administration.
package admin

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/goccy/go-yaml"
	"github.com/urfave/cli/v3"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"github.com/flowline-io/flowbot/internal/store/model"
	"github.com/flowline-io/flowbot/pkg/auth"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/types"
)

const (
	defaultConfigPath = "./flowbot.yaml"
	defaultExpires    = "0d"
	neverExpireYears  = 100
)

// configType holds the database connection section from flowbot.yaml.
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

// AdminCommand returns the admin command group with management subcommands.
func AdminCommand() *cli.Command {
	return &cli.Command{
		Name:  "admin",
		Usage: "admin management tools",
		Commands: []*cli.Command{
			tokenCreateCommand(),
		},
	}
}

// tokenCreateCommand returns the token create subcommand.
func tokenCreateCommand() *cli.Command {
	return &cli.Command{
		Name:  "token",
		Usage: "manage CLI access tokens",
		Commands: []*cli.Command{
			{
				Name:  "create",
				Usage: "create a new CLI access token",
				Description: "Create a new access token for CLI authentication." +
					" Scopes are selected interactively from a numbered list.",
				Flags: []cli.Flag{
					&cli.IntFlag{
						Name:     "id",
						Usage:    "user table row ID",
						Required: true,
					},
					&cli.StringFlag{
						Name:  "expires",
						Value: defaultExpires,
						Usage: "token duration (e.g. 365d, 24h, 30m); 0d means never",
					},
					&cli.StringFlag{
						Name:  "config",
						Value: defaultConfigPath,
						Usage: "config file path",
					},
				},
				Action: tokenCreateAction,
			},
		},
	}
}

// tokenCreateAction creates a CLI access token and writes it to the parameter table.
func tokenCreateAction(_ context.Context, c *cli.Command) error {
	configFile := c.String("config")

	dsn, err := loadDSN(configFile)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	db, err := gorm.Open(mysql.Open(dsn))
	if err != nil {
		flog.Panic("%s", err.Error())
	}

	userID := c.Int("id")

	var user model.User
	if err := db.Where("id = ?", userID).First(&user).Error; err != nil {
		return fmt.Errorf("user not found with id %d: %w", userID, err)
	}

	if user.State == model.UserStateUnknown {
		return fmt.Errorf("user %d (%s) is in unknown state", user.ID, user.Flag)
	}

	_, _ = fmt.Printf("User: name=%s flag=%s\n\n", user.Name, user.Flag)

	scopes, err := selectScopes()
	if err != nil {
		return fmt.Errorf("select scopes: %w", err)
	}

	token := types.Id()

	expiresStr := c.String("expires")
	expiredAt, err := parseExpires(expiresStr)
	if err != nil {
		return fmt.Errorf("parse expires %q: %w", expiresStr, err)
	}

	params := types.KV{
		"uid":    user.Flag,
		"topic":  "",
		"scopes": scopes,
	}

	if err := db.Create(&model.Parameter{
		Flag:      token,
		Params:    model.JSON(params),
		ExpiredAt: expiredAt,
	}).Error; err != nil {
		return fmt.Errorf("create token record: %w", err)
	}

	_, _ = fmt.Printf("\nToken created:\n  %s\n", token)
	return nil
}

// loadDSN reads the MySQL DSN from a flowbot.yaml config file.
func loadDSN(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open config: %w", err)
	}
	defer func() { _ = file.Close() }()

	data, err := io.ReadAll(file)
	if err != nil {
		return "", fmt.Errorf("read config: %w", err)
	}

	var cfg configType
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return "", fmt.Errorf("parse config: %w", err)
	}

	if cfg.StoreConfig.UseAdapter != "mysql" {
		return "", fmt.Errorf("unsupported adapter: %s", cfg.StoreConfig.UseAdapter)
	}
	if cfg.StoreConfig.Adapters.Mysql.DSN == "" {
		return "", fmt.Errorf("mysql DSN is empty")
	}

	return cfg.StoreConfig.Adapters.Mysql.DSN, nil
}

// selectScopes prints available scopes and reads user selection from stdin.
// An empty input defaults to auth.ScopeAdmin.
func selectScopes() ([]string, error) {
	all := auth.AllScopes()

	_, _ = fmt.Println("Available scopes:")
	for i, s := range all {
		_, _ = fmt.Printf("  [%2d] %-30s %s\n", i+1, s.Value, s.Description)
	}

	_, _ = fmt.Print("\nSelect scopes (numbers, comma/space separated, default=admin:*):\n> ")

	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("read input: %w", err)
	}

	line = strings.TrimSpace(line)
	if line == "" {
		return []string{auth.ScopeAdmin}, nil
	}

	var selected []string
	seen := make(map[int]bool)
	for _, part := range strings.FieldsFunc(line, func(r rune) bool {
		return r == ',' || r == ' '
	}) {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		n, err := strconv.Atoi(part)
		if err != nil || n < 1 || n > len(all) {
			_, _ = fmt.Printf("skipping invalid selection: %q\n", part)
			continue
		}
		if seen[n] {
			continue
		}
		seen[n] = true
		selected = append(selected, all[n-1].Value)
	}

	if len(selected) == 0 {
		return nil, fmt.Errorf("no valid scopes selected")
	}

	return selected, nil
}

// parseExpires parses a duration string like "365d", "24h", "30m" into an
// absolute expiration time. The default "0d" means never expires (100 years).
func parseExpires(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" || s == "0" || s == "0d" {
		return time.Now().Add(neverExpireYears * 365 * 24 * time.Hour), nil
	}

	i := strings.IndexFunc(s, func(r rune) bool {
		return !unicode.IsDigit(r)
	})
	if i < 0 {
		return time.Time{}, fmt.Errorf("missing unit (d/h/m)")
	}

	num, err := strconv.Atoi(s[:i])
	if err != nil || num <= 0 {
		return time.Time{}, fmt.Errorf("invalid number: %s", s[:i])
	}
	unit := strings.ToLower(s[i:])

	var dur time.Duration
	switch unit {
	case "d":
		dur = time.Duration(num) * 24 * time.Hour
	case "h":
		dur = time.Duration(num) * time.Hour
	case "m":
		dur = time.Duration(num) * time.Minute
	default:
		return time.Time{}, fmt.Errorf("unknown unit: %s (use d, h, or m)", unit)
	}

	return time.Now().Add(dur), nil
}
