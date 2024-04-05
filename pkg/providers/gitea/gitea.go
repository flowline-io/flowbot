package gitea

import (
	"fmt"
	"github.com/go-resty/resty/v2"
	jsoniter "github.com/json-iterator/go"
	"net/http"
	"time"
)

const (
	ID          = "gitea"
	EndpointKey = "endpoint"
	TokenKey    = "token"
)

type Payload struct {
	Ref        string `json:"ref"`
	Before     string `json:"before"`
	After      string `json:"after"`
	CompareUrl string `json:"compare_url"`
	Commits    []struct {
		Id      string `json:"id"`
		Message string `json:"message"`
		Url     string `json:"url"`
		Author  struct {
			Name     string `json:"name"`
			Email    string `json:"email"`
			Username string `json:"username"`
		} `json:"author"`
		Committer struct {
			Name     string `json:"name"`
			Email    string `json:"email"`
			Username string `json:"username"`
		} `json:"committer"`
		Verification interface{} `json:"verification"`
		Timestamp    time.Time   `json:"timestamp"`
		Added        interface{} `json:"added"`
		Removed      interface{} `json:"removed"`
		Modified     interface{} `json:"modified"`
	} `json:"commits"`
	TotalCommits int `json:"total_commits"`
	HeadCommit   struct {
		Id      string `json:"id"`
		Message string `json:"message"`
		Url     string `json:"url"`
		Author  struct {
			Name     string `json:"name"`
			Email    string `json:"email"`
			Username string `json:"username"`
		} `json:"author"`
		Committer struct {
			Name     string `json:"name"`
			Email    string `json:"email"`
			Username string `json:"username"`
		} `json:"committer"`
		Verification interface{} `json:"verification"`
		Timestamp    time.Time   `json:"timestamp"`
		Added        interface{} `json:"added"`
		Removed      interface{} `json:"removed"`
		Modified     interface{} `json:"modified"`
	} `json:"head_commit"`
	Repository struct {
		Id    int `json:"id"`
		Owner struct {
			Id                int       `json:"id"`
			Login             string    `json:"login"`
			LoginName         string    `json:"login_name"`
			FullName          string    `json:"full_name"`
			Email             string    `json:"email"`
			AvatarUrl         string    `json:"avatar_url"`
			Language          string    `json:"language"`
			IsAdmin           bool      `json:"is_admin"`
			LastLogin         time.Time `json:"last_login"`
			Created           time.Time `json:"created"`
			Restricted        bool      `json:"restricted"`
			Active            bool      `json:"active"`
			ProhibitLogin     bool      `json:"prohibit_login"`
			Location          string    `json:"location"`
			Website           string    `json:"website"`
			Description       string    `json:"description"`
			Visibility        string    `json:"visibility"`
			FollowersCount    int       `json:"followers_count"`
			FollowingCount    int       `json:"following_count"`
			StarredReposCount int       `json:"starred_repos_count"`
			Username          string    `json:"username"`
		} `json:"owner"`
		Name            string      `json:"name"`
		FullName        string      `json:"full_name"`
		Description     string      `json:"description"`
		Empty           bool        `json:"empty"`
		Private         bool        `json:"private"`
		Fork            bool        `json:"fork"`
		Template        bool        `json:"template"`
		Parent          interface{} `json:"parent"`
		Mirror          bool        `json:"mirror"`
		Size            int         `json:"size"`
		Language        string      `json:"language"`
		LanguagesUrl    string      `json:"languages_url"`
		HtmlUrl         string      `json:"html_url"`
		Url             string      `json:"url"`
		Link            string      `json:"link"`
		SshUrl          string      `json:"ssh_url"`
		CloneUrl        string      `json:"clone_url"`
		OriginalUrl     string      `json:"original_url"`
		Website         string      `json:"website"`
		StarsCount      int         `json:"stars_count"`
		ForksCount      int         `json:"forks_count"`
		WatchersCount   int         `json:"watchers_count"`
		OpenIssuesCount int         `json:"open_issues_count"`
		OpenPrCounter   int         `json:"open_pr_counter"`
		ReleaseCounter  int         `json:"release_counter"`
		DefaultBranch   string      `json:"default_branch"`
		Archived        bool        `json:"archived"`
		CreatedAt       time.Time   `json:"created_at"`
		UpdatedAt       time.Time   `json:"updated_at"`
		ArchivedAt      time.Time   `json:"archived_at"`
		Permissions     struct {
			Admin bool `json:"admin"`
			Push  bool `json:"push"`
			Pull  bool `json:"pull"`
		} `json:"permissions"`
		HasIssues       bool `json:"has_issues"`
		InternalTracker struct {
			EnableTimeTracker                bool `json:"enable_time_tracker"`
			AllowOnlyContributorsToTrackTime bool `json:"allow_only_contributors_to_track_time"`
			EnableIssueDependencies          bool `json:"enable_issue_dependencies"`
		} `json:"internal_tracker"`
		HasWiki                       bool        `json:"has_wiki"`
		HasPullRequests               bool        `json:"has_pull_requests"`
		HasProjects                   bool        `json:"has_projects"`
		HasReleases                   bool        `json:"has_releases"`
		HasPackages                   bool        `json:"has_packages"`
		HasActions                    bool        `json:"has_actions"`
		IgnoreWhitespaceConflicts     bool        `json:"ignore_whitespace_conflicts"`
		AllowMergeCommits             bool        `json:"allow_merge_commits"`
		AllowRebase                   bool        `json:"allow_rebase"`
		AllowRebaseExplicit           bool        `json:"allow_rebase_explicit"`
		AllowSquashMerge              bool        `json:"allow_squash_merge"`
		AllowRebaseUpdate             bool        `json:"allow_rebase_update"`
		DefaultDeleteBranchAfterMerge bool        `json:"default_delete_branch_after_merge"`
		DefaultMergeStyle             string      `json:"default_merge_style"`
		DefaultAllowMaintainerEdit    bool        `json:"default_allow_maintainer_edit"`
		AvatarUrl                     string      `json:"avatar_url"`
		Internal                      bool        `json:"internal"`
		MirrorInterval                string      `json:"mirror_interval"`
		MirrorUpdated                 time.Time   `json:"mirror_updated"`
		RepoTransfer                  interface{} `json:"repo_transfer"`
	} `json:"repository"`
	Pusher struct {
		Id                int       `json:"id"`
		Login             string    `json:"login"`
		LoginName         string    `json:"login_name"`
		FullName          string    `json:"full_name"`
		Email             string    `json:"email"`
		AvatarUrl         string    `json:"avatar_url"`
		Language          string    `json:"language"`
		IsAdmin           bool      `json:"is_admin"`
		LastLogin         time.Time `json:"last_login"`
		Created           time.Time `json:"created"`
		Restricted        bool      `json:"restricted"`
		Active            bool      `json:"active"`
		ProhibitLogin     bool      `json:"prohibit_login"`
		Location          string    `json:"location"`
		Website           string    `json:"website"`
		Description       string    `json:"description"`
		Visibility        string    `json:"visibility"`
		FollowersCount    int       `json:"followers_count"`
		FollowingCount    int       `json:"following_count"`
		StarredReposCount int       `json:"starred_repos_count"`
		Username          string    `json:"username"`
	} `json:"pusher"`
	Sender struct {
		Id                int       `json:"id"`
		Login             string    `json:"login"`
		LoginName         string    `json:"login_name"`
		FullName          string    `json:"full_name"`
		Email             string    `json:"email"`
		AvatarUrl         string    `json:"avatar_url"`
		Language          string    `json:"language"`
		IsAdmin           bool      `json:"is_admin"`
		LastLogin         time.Time `json:"last_login"`
		Created           time.Time `json:"created"`
		Restricted        bool      `json:"restricted"`
		Active            bool      `json:"active"`
		ProhibitLogin     bool      `json:"prohibit_login"`
		Location          string    `json:"location"`
		Website           string    `json:"website"`
		Description       string    `json:"description"`
		Visibility        string    `json:"visibility"`
		FollowersCount    int       `json:"followers_count"`
		FollowingCount    int       `json:"following_count"`
		StarredReposCount int       `json:"starred_repos_count"`
		Username          string    `json:"username"`
	} `json:"sender"`
}

type Gitea struct {
	token string
	c     *resty.Client
}

func NewGitea(endpoint, token string) *Gitea {
	v := &Gitea{token: token}
	v.c = resty.New()
	v.c.SetBaseURL(endpoint)
	v.c.SetTimeout(time.Minute)
	v.c.SetAuthToken(token)

	return v
}

func (v *Gitea) GetRepositories() ([]string, error) {
	resp, err := v.c.R().Get("/user/repos")
	if err != nil {
		return nil, err
	}

	if resp.StatusCode() == http.StatusOK {
		var results []string
		var json = jsoniter.ConfigCompatibleWithStandardLibrary
		_ = json.Unmarshal(resp.Body(), &results)
		return results, nil
	} else {
		return nil, fmt.Errorf("%d", resp.StatusCode())
	}

}
