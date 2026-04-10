package github

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGithub_GetAuthenticatedUser(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/user", r.URL.Path)
		assert.Equal(t, "token test_token", r.Header.Get("Authorization"))
		assert.Equal(t, JSONAccept, r.Header.Get("Accept"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"login": "testuser",
			"id": 12345,
			"name": "Test User",
			"followers": 10,
			"following": 5
		}`))
	}))
	defer server.Close()

	client := NewGithub("id", "secret", "redirect", "test_token")
	client.c.SetBaseURL(server.URL)

	user, err := client.GetAuthenticatedUser()
	require.NoError(t, err)
	require.NotNil(t, user)
	assert.Equal(t, "testuser", *user.Login)
	assert.Equal(t, int64(12345), *user.ID)
}

func TestGithub_GetUser(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/users/octocat", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"login": "octocat",
			"id": 583231,
			"name": "The Octocat"
		}`))
	}))
	defer server.Close()

	client := NewGithub("id", "secret", "redirect", "test_token")
	client.c.SetBaseURL(server.URL)

	user, err := client.GetUser("octocat")
	require.NoError(t, err)
	require.NotNil(t, user)
	assert.Equal(t, "octocat", *user.Login)
}

func TestGithub_GetAuthenticatedUser_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	client := NewGithub("id", "secret", "redirect", "bad_token")
	client.c.SetBaseURL(server.URL)

	user, err := client.GetAuthenticatedUser()
	assert.Error(t, err)
	assert.Nil(t, user)
}

func TestGithub_GetRepository(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/repos/owner/repo", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"id": 123456,
			"name": "repo",
			"full_name": "owner/repo",
			"stargazers_count": 100
		}`))
	}))
	defer server.Close()

	client := NewGithub("id", "secret", "redirect", "test_token")
	client.c.SetBaseURL(server.URL)

	repo, err := client.GetRepository("owner", "repo")
	require.NoError(t, err)
	require.NotNil(t, repo)
	assert.Equal(t, "repo", *repo.Name)
	assert.Equal(t, int64(123456), *repo.ID)
}

func TestGithub_GetRepository_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := NewGithub("id", "secret", "redirect", "test_token")
	client.c.SetBaseURL(server.URL)

	repo, err := client.GetRepository("owner", "nonexistent")
	assert.Error(t, err)
	assert.Nil(t, repo)
}

func TestGithub_CreateIssue(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/repos/owner/repo/issues", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{
			"id": 1,
			"number": 42,
			"title": "Test Issue",
			"html_url": "https://github.com/owner/repo/issues/42"
		}`))
	}))
	defer server.Close()

	client := NewGithub("id", "secret", "redirect", "test_token")
	client.c.SetBaseURL(server.URL)

	issue, err := client.CreateIssue("owner", "repo", Issue{Title: strPtr("Test Issue")})
	require.NoError(t, err)
	require.NotNil(t, issue)
	assert.Equal(t, 42, *issue.Number)
	assert.Equal(t, "Test Issue", *issue.Title)
}

func TestGithub_GetNotifications(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/notifications", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[
			{"id": "1", "reason": "mention", "unread": true,
			 "subject": {"title": "PR Review", "type": "PullRequest"},
			 "repository": {"id": 123, "name": "repo", "full_name": "owner/repo"}}
		]`))
	}))
	defer server.Close()

	client := NewGithub("id", "secret", "redirect", "test_token")
	client.c.SetBaseURL(server.URL)

	notifications, err := client.GetNotifications()
	require.NoError(t, err)
	require.Len(t, notifications, 1)
	assert.Equal(t, "1", *notifications[0].ID)
}

func TestGithub_GetReleases(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/repos/owner/repo/releases", r.URL.Path)
		assert.Equal(t, "1", r.URL.Query().Get("page"))
		assert.Equal(t, "10", r.URL.Query().Get("per_page"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[
			{"tag_name": "v1.0.0", "name": "Release 1.0.0"}
		]`))
	}))
	defer server.Close()

	client := NewGithub("id", "secret", "redirect", "test_token")
	client.c.SetBaseURL(server.URL)

	releases, err := client.GetReleases("owner", "repo", 1, 10)
	require.NoError(t, err)
	require.Len(t, releases, 1)
	assert.Equal(t, "v1.0.0", *releases[0].TagName)
}

func TestGithub_GetFollowers(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/user/followers", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[
			{"login": "follower1", "id": 1},
			{"login": "follower2", "id": 2}
		]`))
	}))
	defer server.Close()

	client := NewGithub("id", "secret", "redirect", "test_token")
	client.c.SetBaseURL(server.URL)

	followers, err := client.GetFollowers()
	require.NoError(t, err)
	require.Len(t, followers, 2)
	assert.Equal(t, "follower1", *followers[0].Login)
}

func TestGithub_GetStarred(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/users/octocat/starred", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[
			{"id": 1, "name": "repo1", "full_name": "owner/repo1"}
		]`))
	}))
	defer server.Close()

	client := NewGithub("id", "secret", "redirect", "test_token")
	client.c.SetBaseURL(server.URL)

	repos, err := client.GetStarred("octocat")
	require.NoError(t, err)
	require.Len(t, repos, 1)
	assert.Equal(t, "repo1", *repos[0].Name)
}

func TestGithub_GetUserProjects(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/users/octocat/projects", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[
			{"id": 1, "name": "Project1"}
		]`))
	}))
	defer server.Close()

	client := NewGithub("id", "secret", "redirect", "test_token")
	client.c.SetBaseURL(server.URL)

	projects, err := client.GetUserProjects("octocat")
	require.NoError(t, err)
	require.Len(t, projects, 1)
	assert.Equal(t, "Project1", *projects[0].Name)
}

func TestGithub_GetProjectColumns(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/projects/1/columns", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[
			{"id": 10, "name": "To Do"}
		]`))
	}))
	defer server.Close()

	client := NewGithub("id", "secret", "redirect", "test_token")
	client.c.SetBaseURL(server.URL)

	columns, err := client.GetProjectColumns(1)
	require.NoError(t, err)
	require.Len(t, columns, 1)
	assert.Equal(t, "To Do", *columns[0].Name)
}

func TestGithub_CreateCard(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/projects/columns/10/cards", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{
			"id": 100,
			"note": "My card"
		}`))
	}))
	defer server.Close()

	client := NewGithub("id", "secret", "redirect", "test_token")
	client.c.SetBaseURL(server.URL)

	card, err := client.CreateCard(10, ProjectCard{Note: strPtr("My card")})
	require.NoError(t, err)
	require.NotNil(t, card)
	assert.Equal(t, int64(100), *card.ID)
	assert.Equal(t, "My card", *card.Note)
}

func strPtr(s string) *string { return &s }
