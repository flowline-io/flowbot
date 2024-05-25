package gitea

import "time"

type Issue struct {
	Action string `json:"action"` // opened, closed, reopened, label_updated, assigned
	Number int    `json:"number"`
	Issue  struct {
		Id      int    `json:"id"`
		Url     string `json:"url"`
		HtmlUrl string `json:"html_url"`
		Number  int    `json:"number"`
		User    struct {
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
		} `json:"user"`
		OriginalAuthor   string        `json:"original_author"`
		OriginalAuthorId int           `json:"original_author_id"`
		Title            string        `json:"title"`
		Body             string        `json:"body"`
		Ref              string        `json:"ref"`
		Assets           []interface{} `json:"assets"`
		Labels           []interface{} `json:"labels"`
		Milestone        interface{}   `json:"milestone"`
		Assignee         interface{}   `json:"assignee"`
		Assignees        interface{}   `json:"assignees"`
		State            string        `json:"state"`
		IsLocked         bool          `json:"is_locked"`
		Comments         int           `json:"comments"`
		CreatedAt        time.Time     `json:"created_at"`
		UpdatedAt        time.Time     `json:"updated_at"`
		ClosedAt         interface{}   `json:"closed_at"`
		DueDate          interface{}   `json:"due_date"`
		PullRequest      interface{}   `json:"pull_request"`
		Repository       struct {
			Id       int    `json:"id"`
			Name     string `json:"name"`
			Owner    string `json:"owner"`
			FullName string `json:"full_name"`
		} `json:"repository"`
		PinOrder int `json:"pin_order"`
	} `json:"issue"`
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
	CommitId string `json:"commit_id"`
}
