package api

import (
	"encoding/json"
	"testing"

	"github.com/flowline-io/flowbot/pkg/types/admin"
)

func TestAPIResponseParsing(t *testing.T) {
	jsonData := `{"status":"ok","data":{"id":1,"name":"test"},"retcode":"","message":""}`

	var resp APIResponse
	err := json.Unmarshal([]byte(jsonData), &resp)
	if err != nil {
		t.Fatalf("Failed to parse APIResponse: %v", err)
	}

	if resp.Status != "ok" {
		t.Errorf("Status = %s, want ok", resp.Status)
	}
	if string(resp.Data) != `{"id":1,"name":"test"}` {
		t.Errorf("Data = %s, want {\"id\":1,\"name\":\"test\"}", string(resp.Data))
	}
}

func TestAPIResponseError(t *testing.T) {
	jsonData := `{"status":"error","retcode":"AUTH_FAILED","message":"Invalid token"}`

	var resp APIResponse
	err := json.Unmarshal([]byte(jsonData), &resp)
	if err != nil {
		t.Fatalf("Failed to parse APIResponse: %v", err)
	}

	if resp.Status != "error" {
		t.Errorf("Status = %s, want error", resp.Status)
	}
	if resp.RetCode != "AUTH_FAILED" {
		t.Errorf("RetCode = %s, want AUTH_FAILED", resp.RetCode)
	}
	if resp.Message != "Invalid token" {
		t.Errorf("Message = %s, want Invalid token", resp.Message)
	}
}

func TestUserTypesParsing(t *testing.T) {
	jsonData := `{"id":1,"uid":"abc123","name":"Test User","email":"test@example.com","role":"admin","status":"active","platform":"slack","created_at":"2024-01-01T00:00:00Z","updated_at":"2024-01-02T00:00:00Z"}`

	var user admin.User
	err := json.Unmarshal([]byte(jsonData), &user)
	if err != nil {
		t.Fatalf("Failed to parse User: %v", err)
	}

	if user.ID != 1 {
		t.Errorf("ID = %d, want 1", user.ID)
	}
	if user.Name != "Test User" {
		t.Errorf("Name = %s, want Test User", user.Name)
	}
	if user.Role != admin.RoleAdmin {
		t.Errorf("Role = %s, want admin", user.Role)
	}
}

func TestLogEntryParsing(t *testing.T) {
	jsonData := `{"id":123,"level":"info","message":"Test log message","source":"server","timestamp":"2024-01-01T12:00:00Z"}`

	var entry admin.LogEntry
	err := json.Unmarshal([]byte(jsonData), &entry)
	if err != nil {
		t.Fatalf("Failed to parse LogEntry: %v", err)
	}

	if entry.ID != 123 {
		t.Errorf("ID = %d, want 123", entry.ID)
	}
	if entry.Level != admin.LogLevelInfo {
		t.Errorf("Level = %s, want info", entry.Level)
	}
	if entry.Message != "Test log message" {
		t.Errorf("Message = %s, want Test log message", entry.Message)
	}
}

func TestWorkflowParsing(t *testing.T) {
	jsonData := `{"id":1,"name":"Test Workflow","description":"A test workflow","status":"pending","created_at":"2024-01-01T00:00:00Z","updated_at":"2024-01-02T00:00:00Z"}`

	var workflow admin.Workflow
	err := json.Unmarshal([]byte(jsonData), &workflow)
	if err != nil {
		t.Fatalf("Failed to parse Workflow: %v", err)
	}

	if workflow.ID != 1 {
		t.Errorf("ID = %d, want 1", workflow.ID)
	}
	if workflow.Name != "Test Workflow" {
		t.Errorf("Name = %s, want Test Workflow", workflow.Name)
	}
	if workflow.Status != admin.WorkflowPending {
		t.Errorf("Status = %s, want pending", workflow.Status)
	}
}

func TestBotInfoParsing(t *testing.T) {
	jsonData := `{"name":"test-bot","enabled":true,"description":"A test bot","commands":["cmd1","cmd2"],"has_form":true,"has_cron":false,"has_webhook":true}`

	var bot admin.BotInfo
	err := json.Unmarshal([]byte(jsonData), &bot)
	if err != nil {
		t.Fatalf("Failed to parse BotInfo: %v", err)
	}

	if bot.Name != "test-bot" {
		t.Errorf("Name = %s, want test-bot", bot.Name)
	}
	if !bot.Enabled {
		t.Errorf("Enabled = %v, want true", bot.Enabled)
	}
	if len(bot.Commands) != 2 {
		t.Errorf("Commands length = %d, want 2", len(bot.Commands))
	}
}

func TestListResponseParsing(t *testing.T) {
	jsonData := `{"items":[{"id":1,"uid":"","name":"item1","email":"","role":"admin","status":"active","platform":"","created_at":"2024-01-01T00:00:00Z","updated_at":"2024-01-01T00:00:00Z"}],"total":10,"page":1,"page_size":10,"total_pages":1}`

	var resp admin.ListResponse[admin.User]
	err := json.Unmarshal([]byte(jsonData), &resp)
	if err != nil {
		t.Fatalf("Failed to parse ListResponse: %v", err)
	}

	if resp.Total != 10 {
		t.Errorf("Total = %d, want 10", resp.Total)
	}
	if resp.TotalPages != 1 {
		t.Errorf("TotalPages = %d, want 1", resp.TotalPages)
	}
	if len(resp.Items) != 1 {
		t.Errorf("Items length = %d, want 1", len(resp.Items))
	}
}

func TestContainerParsing(t *testing.T) {
	jsonData := `{"id":1,"name":"web-server","status":"running","created_at":"2024-01-01T00:00:00Z"}`

	var container admin.Container
	err := json.Unmarshal([]byte(jsonData), &container)
	if err != nil {
		t.Fatalf("Failed to parse Container: %v", err)
	}

	if container.ID != 1 {
		t.Errorf("ID = %d, want 1", container.ID)
	}
	if container.Name != "web-server" {
		t.Errorf("Name = %s, want web-server", container.Name)
	}
	if container.Status != admin.ContainerRunning {
		t.Errorf("Status = %s, want running", container.Status)
	}
}

func TestSettingsParsing(t *testing.T) {
	jsonData := `{"site_name":"Flowbot","logo_url":"https://example.com/logo.png","seo_description":"A chatbot framework","max_upload_size":10485760}`

	var settings admin.Settings
	err := json.Unmarshal([]byte(jsonData), &settings)
	if err != nil {
		t.Fatalf("Failed to parse Settings: %v", err)
	}

	if settings.SiteName != "Flowbot" {
		t.Errorf("SiteName = %s, want Flowbot", settings.SiteName)
	}
	if settings.LogoURL != "https://example.com/logo.png" {
		t.Errorf("LogoURL = %s, want https://example.com/logo.png", settings.LogoURL)
	}
	if settings.MaxUploadSize != 10485760 {
		t.Errorf("MaxUploadSize = %d, want 10485760", settings.MaxUploadSize)
	}
}
