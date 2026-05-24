package github

import (
	"testing"

	"github.com/flowline-io/flowbot/pkg/ability"
	githubsvc "github.com/flowline-io/flowbot/pkg/ability/github"
	"github.com/flowline-io/flowbot/pkg/ability/conformance"
	provider "github.com/flowline-io/flowbot/pkg/providers/github"
)

// newConformanceService wraps an Adapter to satisfy the conformance ServiceFactory contract.
// It constructs a fakeClient from the conformance Config and returns the adapter.
func newConformanceService(t *testing.T, cfg githubsvc.Config) githubsvc.Service {
	t.Helper()
	c := &fakeClient{
		user:             cfgToUser(cfg.User),
		userErr:          cfg.UserErr,
		userByLogin:      cfgToUser(cfg.UserByLogin),
		userByLoginErr:   cfg.UserByLoginErr,
		repo:             cfgToRepo(cfg.Repo),
		repoErr:          cfg.RepoErr,
		issues:           cfgToIssues(cfg.Issues),
		issuesErr:        cfg.IssuesErr,
		diff:             cfgToDiff(cfg.Diff),
		diffErr:          cfg.DiffErr,
		fileContent:      cfg.FileContent,
		fileContentErr:   cfg.FileContentErr,
		notifications:    cfgToNotifications(cfg.Notifications),
		notificationsErr: cfg.NotificationsErr,
		releases:         cfgToReleases(cfg.Releases),
		releasesErr:      cfg.ReleasesErr,
	}
	a, ok := NewWithClient(c).(*Adapter)
	if !ok {
		t.Fatal("unexpected type")
	}
	a.cursorSecret = conformance.CursorSecret
	a.now = conformance.TestTime
	return a
}

// TestGithubConformance runs the standard GitHub capability conformance suite.
func TestGithubConformance(t *testing.T) {
	githubsvc.RunGithubConformance(t, func(_ *testing.T, cfg githubsvc.Config) githubsvc.Service {
		return newConformanceService(t, cfg)
	})
}

func cfgToUser(user *ability.ForgeUser) *provider.User {
	if user == nil {
		return nil
	}
	id := user.ID
	login := user.UserName
	email := user.Email
	avatarURL := user.AvatarURL
	return &provider.User{
		ID:        &id,
		Login:     &login,
		Email:     &email,
		AvatarURL: &avatarURL,
	}
}

func cfgToRepo(repo *ability.ForgeRepo) *provider.Repository {
	if repo == nil {
		return nil
	}
	id := repo.ID
	name := repo.Name
	fullName := repo.FullName
	owner := repo.Owner
	desc := repo.Description
	private := repo.Private
	htmlURL := repo.HTMLURL
	cloneURL := repo.CloneURL
	return &provider.Repository{
		ID:          &id,
		Name:        &name,
		FullName:    &fullName,
		Owner:       &provider.User{Login: &owner},
		Description: &desc,
		Private:     &private,
		HTMLURL:     &htmlURL,
		CloneURL:    &cloneURL,
	}
}

func cfgToIssues(issues []*ability.ForgeIssue) []*provider.Issue {
	if issues == nil {
		return nil
	}
	result := make([]*provider.Issue, 0, len(issues))
	for _, iss := range issues {
		id := iss.ID
		number := int(iss.Index)
		title := iss.Title
		body := iss.Body
		state := iss.State
		htmlURL := iss.HTMLURL
		author := iss.Author
		result = append(result, &provider.Issue{
			ID:      &id,
			Number:  &number,
			Title:   &title,
			Body:    &body,
			State:   &state,
			HTMLURL: &htmlURL,
			User:    &provider.User{Login: &author},
			Repository: &provider.Repository{Name: strPtr("repo")},
		})
	}
	return result
}

func cfgToDiff(diff *ability.ForgeCommitDiff) *provider.CommitDiff {
	if diff == nil {
		return nil
	}
	return &provider.CommitDiff{
		CommitID:      diff.CommitID,
		CommitMessage: diff.CommitMessage,
		Files:         diff.Files,
		DiffContent:   diff.DiffContent,
	}
}

func cfgToNotifications(notifications []*ability.Notification) []*provider.Notification {
	if notifications == nil {
		return nil
	}
	result := make([]*provider.Notification, 0, len(notifications))
	for _, n := range notifications {
		id := n.ID
		reason := n.Reason
		unread := n.Unread
		subject := n.Subject
		repoName := n.RepoName
		result = append(result, &provider.Notification{
			ID:     &id,
			Reason: &reason,
			Unread: &unread,
			Subject: &provider.Subject{
				Title: &subject,
			},
			Repository: &provider.Repository{
				FullName: &repoName,
			},
		})
	}
	return result
}

func cfgToReleases(releases []*ability.Release) []*provider.RepositoryRelease {
	if releases == nil {
		return nil
	}
	result := make([]*provider.RepositoryRelease, 0, len(releases))
	for _, r := range releases {
		id := r.ID
		tagName := r.TagName
		name := r.Name
		body := r.Body
		draft := r.Draft
		prerelease := r.Prerelease
		htmlURL := r.HTMLURL
		result = append(result, &provider.RepositoryRelease{
			ID:         &id,
			TagName:    &tagName,
			Name:       &name,
			Body:       &body,
			Draft:      &draft,
			Prerelease: &prerelease,
			HTMLURL:    &htmlURL,
		})
	}
	return result
}
