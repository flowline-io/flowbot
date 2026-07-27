package admin

import (
	"bufio"
	"database/sql"
	"fmt"
	"os"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/spf13/cobra"

	"github.com/flowline-io/flowbot/pkg/webauth"
)

func authCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "manage web UI authentication",
	}
	cmd.AddCommand(reset2FACommand())
	return cmd
}

func reset2FACommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reset-2fa",
		Short: "clear TOTP for a web account (forces re-enroll on next login)",
		RunE:  reset2FAAction,
	}
	cmd.Flags().String("username", "", "web account username")
	_ = cmd.MarkFlagRequired("username")
	cmd.Flags().String("config", defaultConfigPath, "config file path")
	cmd.Flags().Bool("yes", false, "skip interactive confirmation")
	return cmd
}

func reset2FAAction(cmd *cobra.Command, _ []string) error {
	username, _ := cmd.Flags().GetString("username")
	username = strings.TrimSpace(username)
	if username == "" {
		return fmt.Errorf("username is required")
	}
	yes, _ := cmd.Flags().GetBool("yes")
	if err := confirmReset2FA(cmd, username, yes); err != nil {
		return err
	}

	configFile, _ := cmd.Flags().GetString("config")
	db, err := openAdminDB(configFile)
	if err != nil {
		return err
	}
	defer func() { _ = db.Close() }()

	if err := clearWebAccountTOTP(db, username); err != nil {
		return err
	}
	deleted, err := revokeWebSessions(db, webauth.UIDForUsername(username))
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "TOTP reset for %q; revoked %d web session(s). Next login must re-enroll 2FA.\n", username, deleted)
	return nil
}

func confirmReset2FA(cmd *cobra.Command, username string, yes bool) error {
	if yes {
		return nil
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Clear TOTP and backup codes for %q? Type the username to confirm: ", username)
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("read confirmation: %w", err)
	}
	if strings.TrimSpace(line) != username {
		return fmt.Errorf("confirmation mismatch; aborted")
	}
	return nil
}

func openAdminDB(configFile string) (*sql.DB, error) {
	dsn, err := loadDSN(configFile)
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}
	return db, nil
}

func clearWebAccountTOTP(db *sql.DB, username string) error {
	res, err := db.Exec(`
		UPDATE web_accounts
		SET totp_secret_ciphertext = NULL,
		    totp_secret_nonce = NULL,
		    totp_enabled = false,
		    totp_last_step = 0,
		    backup_code_hashes = $1,
		    updated_at = NOW()
		WHERE username = $2
	`, "[]", username)
	if err != nil {
		return fmt.Errorf("reset totp: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("web account %q not found", username)
	}
	return nil
}

func revokeWebSessions(db *sql.DB, uid string) (int, error) {
	rows, err := db.Query(`SELECT id, params FROM parameter`)
	if err != nil {
		return 0, fmt.Errorf("list sessions: %w", err)
	}
	defer func() { _ = rows.Close() }()
	deleted := 0
	for rows.Next() {
		var id int64
		var paramsRaw []byte
		if err := rows.Scan(&id, &paramsRaw); err != nil {
			return deleted, err
		}
		if !isWebSessionForUID(paramsRaw, uid) {
			continue
		}
		if _, err := db.Exec(`DELETE FROM parameter WHERE id = $1`, id); err != nil {
			return deleted, fmt.Errorf("delete session %d: %w", id, err)
		}
		deleted++
	}
	return deleted, rows.Err()
}

func isWebSessionForUID(paramsRaw []byte, uid string) bool {
	var params map[string]any
	if err := sonic.Unmarshal(paramsRaw, &params); err != nil {
		return false
	}
	uidVal, ok := params["uid"].(string)
	if !ok || uidVal != uid {
		return false
	}
	topic, topicOK := params["topic"].(string)
	if !topicOK {
		topic = ""
	}
	kind, kindOK := params["kind"].(string)
	if !kindOK {
		kind = ""
	}
	return topic == "web" || kind != ""
}
