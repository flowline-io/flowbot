package github

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTokenResponse_Unmarshal(t *testing.T) {
	data := `{
		"access_token": "gho_test_token",
		"scope": "repo,user",
		"token_type": "bearer"
	}`

	var token TokenResponse
	err := json.Unmarshal([]byte(data), &token)
	require.NoError(t, err)

	assert.Equal(t, "gho_test_token", token.AccessToken)
	assert.Equal(t, "repo,user", token.Scope)
	assert.Equal(t, "bearer", token.TokenType)
}

func TestUser_Unmarshal(t *testing.T) {
	data := `{
		"login": "testuser",
		"id": 12345,
		"node_id": "MDQ6VXNlcjEyMzQ1",
		"avatar_url": "https://avatars.githubusercontent.com/u/12345",
		"html_url": "https://github.com/testuser",
		"name": "Test User",
		"company": "Test Corp",
		"blog": "https://testuser.dev",
		"location": "San Francisco",
		"email": "test@example.com",
		"bio": "Software developer",
		"public_repos": 50,
		"public_gists": 10,
		"followers": 100,
		"following": 50,
		"created_at": "2020-01-01T00:00:00Z",
		"updated_at": "2024-01-01T00:00:00Z"
	}`

	var user User
	err := json.Unmarshal([]byte(data), &user)
	require.NoError(t, err)

	assert.Equal(t, "testuser", *user.Login)
	assert.Equal(t, int64(12345), *user.ID)
	assert.Equal(t, "Test User", *user.Name)
	assert.Equal(t, "test@example.com", *user.Email)
	assert.Equal(t, 50, *user.PublicRepos)
	assert.Equal(t, 100, *user.Followers)
}

func TestRepository_Unmarshal(t *testing.T) {
	data := `{
		"id": 123456789,
		"node_id": "MDEwOlJlcG9zaXRvcnkxMjM0NTY3ODk=",
		"name": "test-repo",
		"full_name": "testuser/test-repo",
		"description": "A test repository",
		"private": false,
		"fork": false,
		"html_url": "https://github.com/testuser/test-repo",
		"language": "Go",
		"forks_count": 10,
		"stargazers_count": 100,
		"watchers_count": 100,
		"open_issues_count": 5,
		"default_branch": "main",
		"topics": ["go", "testing", "example"]
	}`

	var repo Repository
	err := json.Unmarshal([]byte(data), &repo)
	require.NoError(t, err)

	assert.Equal(t, int64(123456789), *repo.ID)
	assert.Equal(t, "test-repo", *repo.Name)
	assert.Equal(t, "testuser/test-repo", *repo.FullName)
	assert.Equal(t, "A test repository", *repo.Description)
	assert.Equal(t, "Go", *repo.Language)
	assert.Equal(t, 100, *repo.StargazersCount)
	assert.Len(t, repo.Topics, 3)
}

func TestIssue_Unmarshal(t *testing.T) {
	data := `{
		"id": 123456,
		"node_id": "MDU6SXNzdWUxMjM0NTY=",
		"number": 42,
		"title": "Test Issue",
		"body": "This is a test issue",
		"state": "open",
		"locked": false,
		"user": {
			"login": "testuser",
			"id": 12345
		},
		"labels": [
			{
				"id": 123,
				"name": "bug",
				"color": "d73a4a"
			}
		],
		"comments": 3,
		"created_at": "2024-01-01T00:00:00Z",
		"updated_at": "2024-01-02T00:00:00Z"
	}`

	var issue Issue
	err := json.Unmarshal([]byte(data), &issue)
	require.NoError(t, err)

	assert.Equal(t, int64(123456), *issue.ID)
	assert.Equal(t, 42, *issue.Number)
	assert.Equal(t, "Test Issue", *issue.Title)
	assert.Equal(t, "open", *issue.State)
	assert.Equal(t, "testuser", *issue.User.Login)
	assert.Len(t, issue.Labels, 1)
}

func TestLabel_Unmarshal(t *testing.T) {
	data := `{
		"id": 123,
		"node_id": "MDU6TGFiZWwxMjM=",
		"name": "bug",
		"description": "Something isn't working",
		"color": "d73a4a",
		"default": true
	}`

	var label Label
	err := json.Unmarshal([]byte(data), &label)
	require.NoError(t, err)

	assert.Equal(t, int64(123), *label.ID)
	assert.Equal(t, "bug", *label.Name)
	assert.Equal(t, "d73a4a", *label.Color)
	assert.True(t, *label.Default)
}

func TestProject_Unmarshal(t *testing.T) {
	data := `{
		"id": 1234567,
		"node_id": "MDc6UHJvamVjdDEyMzQ1Njc=",
		"name": "Test Project",
		"body": "Project description",
		"number": 1,
		"state": "open",
		"html_url": "https://github.com/testuser/test-repo/projects/1"
	}`

	var project Project
	err := json.Unmarshal([]byte(data), &project)
	require.NoError(t, err)

	assert.Equal(t, int64(1234567), *project.ID)
	assert.Equal(t, "Test Project", *project.Name)
	assert.Equal(t, "open", *project.State)
}

func TestProjectColumn_Unmarshal(t *testing.T) {
	data := `{
		"id": 12345678,
		"node_id": "MDEzOlByb2plY3RDb2x1bW4xMjM0NTY3OA==",
		"name": "To Do",
		"project_url": "https://api.github.com/projects/1234567"
	}`

	var column ProjectColumn
	err := json.Unmarshal([]byte(data), &column)
	require.NoError(t, err)

	assert.Equal(t, int64(12345678), *column.ID)
	assert.Equal(t, "To Do", *column.Name)
}

func TestProjectCard_Unmarshal(t *testing.T) {
	data := `{
		"id": 123456789,
		"node_id": "MDExOlByb2plY3RDYXJkMTIzNDU2Nzg5",
		"note": "Test card note",
		"archived": false,
		"column_id": 12345678
	}`

	var card ProjectCard
	err := json.Unmarshal([]byte(data), &card)
	require.NoError(t, err)

	assert.Equal(t, int64(123456789), *card.ID)
	assert.Equal(t, "Test card note", *card.Note)
	assert.Equal(t, int64(12345678), *card.ColumnID)
}

func TestNotification_Unmarshal(t *testing.T) {
	data := `{
		"id": "123456789",
		"repository": {
			"id": 123456,
			"name": "test-repo",
			"full_name": "testuser/test-repo"
		},
		"subject": {
			"title": "Test PR",
			"type": "PullRequest"
		},
		"reason": "mention",
		"unread": true,
		"updated_at": "2024-01-01T00:00:00Z"
	}`

	var notification Notification
	err := json.Unmarshal([]byte(data), &notification)
	require.NoError(t, err)

	assert.Equal(t, "123456789", *notification.ID)
	assert.Equal(t, "test-repo", *notification.Repository.Name)
	assert.Equal(t, "mention", *notification.Reason)
	assert.True(t, *notification.Unread)
}

func TestRepositoryRelease_Unmarshal(t *testing.T) {
	data := `{
		"id": 123456,
		"tag_name": "v1.0.0",
		"name": "Release 1.0.0",
		"body": "Release notes",
		"draft": false,
		"prerelease": false,
		"created_at": "2024-01-01T00:00:00Z",
		"published_at": "2024-01-01T12:00:00Z",
		"html_url": "https://github.com/testuser/test-repo/releases/tag/v1.0.0"
	}`

	var release RepositoryRelease
	err := json.Unmarshal([]byte(data), &release)
	require.NoError(t, err)

	assert.Equal(t, "v1.0.0", *release.TagName)
	assert.Equal(t, "Release 1.0.0", *release.Name)
	assert.Equal(t, "Release notes", *release.Body)
	assert.False(t, *release.Draft)
}

func TestPackageWebhook_Unmarshal(t *testing.T) {
	data := `{
		"action": "published",
		"package": {
			"id": 12345,
			"name": "test-package",
			"namespace": "testuser",
			"description": "A test package",
			"ecosystem": "docker",
			"package_type": "container",
			"html_url": "https://github.com/testuser/test-repo/pkgs/container/test-package",
			"package_version": {
				"id": 67890,
				"version": "1.0.0",
				"name": "test-package:1.0.0",
				"description": "Version 1.0.0",
				"container_metadata": {
					"tag": {
						"name": "1.0.0",
						"digest": "sha256:abc123"
					}
				}
			}
		}
	}`

	var webhook PackageWebhook
	err := json.Unmarshal([]byte(data), &webhook)
	require.NoError(t, err)

	assert.Equal(t, "published", webhook.Action)
	assert.Equal(t, "test-package", webhook.Package.Name)
	assert.Equal(t, "docker", webhook.Package.Ecosystem)
	assert.Equal(t, "1.0.0", webhook.Package.PackageVersion.Version)
	assert.Equal(t, "sha256:abc123", webhook.Package.PackageVersion.ContainerMetadata.Tag.Digest)
}

func TestMilestone_Unmarshal(t *testing.T) {
	data := `{
		"id": 12345,
		"number": 1,
		"title": "v1.0",
		"description": "First release",
		"state": "open",
		"open_issues": 5,
		"closed_issues": 3,
		"created_at": "2024-01-01T00:00:00Z",
		"due_on": "2024-06-01T00:00:00Z"
	}`

	var milestone Milestone
	err := json.Unmarshal([]byte(data), &milestone)
	require.NoError(t, err)

	assert.Equal(t, int64(12345), *milestone.ID)
	assert.Equal(t, "v1.0", *milestone.Title)
	assert.Equal(t, 5, *milestone.OpenIssues)
}

func TestReactions_Unmarshal(t *testing.T) {
	data := `{
		"total_count": 10,
		"+1": 5,
		"-1": 1,
		"laugh": 2,
		"confused": 0,
		"heart": 1,
		"hooray": 1,
		"rocket": 0,
		"eyes": 0
	}`

	var reactions Reactions
	err := json.Unmarshal([]byte(data), &reactions)
	require.NoError(t, err)

	assert.Equal(t, 10, *reactions.TotalCount)
	assert.Equal(t, 5, *reactions.PlusOne)
	assert.Equal(t, 1, *reactions.MinusOne)
	assert.Equal(t, 2, *reactions.Laugh)
}

func TestReleaseAsset_Unmarshal(t *testing.T) {
	data := `{
		"id": 12345,
		"name": "test-asset.zip",
		"content_type": "application/zip",
		"size": 1024000,
		"download_count": 100,
		"browser_download_url": "https://github.com/testuser/test-repo/releases/download/v1.0.0/test-asset.zip",
		"created_at": "2024-01-01T00:00:00Z",
		"updated_at": "2024-01-02T00:00:00Z"
	}`

	var asset ReleaseAsset
	err := json.Unmarshal([]byte(data), &asset)
	require.NoError(t, err)

	assert.Equal(t, int64(12345), *asset.ID)
	assert.Equal(t, "test-asset.zip", *asset.Name)
	assert.Equal(t, "application/zip", *asset.ContentType)
	assert.Equal(t, 1024000, *asset.Size)
}

func TestConstants(t *testing.T) {
	assert.Equal(t, "github", ID)
	assert.Equal(t, "id", ClientIdKey)
	assert.Equal(t, "secret", ClientSecretKey)
	assert.Equal(t, "application/vnd.github.v3+json", JSONAccept)
}

func TestGithub_Constructor(t *testing.T) {
	clientId := "test_client_id"
	clientSecret := "test_client_secret"
	redirectURI := "https://example.com/callback"
	accessToken := "test_access_token"

	github := NewGithub(clientId, clientSecret, redirectURI, accessToken)

	assert.NotNil(t, github)
	assert.Equal(t, clientId, github.clientId)
	assert.Equal(t, clientSecret, github.clientSecret)
	assert.Equal(t, redirectURI, github.redirectURI)
	assert.Equal(t, accessToken, github.accessToken)
}

func TestGithub_GetAuthorizeURL(t *testing.T) {
	github := NewGithub("client_id", "secret", "https://example.com/callback", "")
	url := github.GetAuthorizeURL()

	assert.Contains(t, url, "https://github.com/login/oauth/authorize")
	assert.Contains(t, url, "client_id=client_id")
	assert.Contains(t, url, "redirect_uri=https://example.com/callback")
	assert.Contains(t, url, "scope=repo")
}

func TestGithub_Redirect(t *testing.T) {
	github := NewGithub("client_id", "secret", "https://example.com/callback", "")
	url, err := github.Redirect(nil)

	assert.NoError(t, err)
	assert.Contains(t, url, "github.com/login/oauth/authorize")
}

func TestTimePointer(t *testing.T) {
	now := time.Now()
	data := `{"created_at": "` + now.Format(time.RFC3339) + `"}`

	var user User
	err := json.Unmarshal([]byte(data), &user)
	require.NoError(t, err)

	assert.NotNil(t, user.CreatedAt)
	assert.WithinDuration(t, now, *user.CreatedAt, time.Second)
}
