package drone

type Build struct {
	ID           int64   `json:"id"`
	RepoID       int64   `json:"repo_id"`
	Number       int     `json:"number"`
	Status       string  `json:"status"`
	Event        string  `json:"event"`
	Action       string  `json:"action"`
	Link         string  `json:"link"`
	Message      string  `json:"message"`
	Before       string  `json:"before"`
	After        string  `json:"after"`
	Ref          string  `json:"ref"`
	SourceRepo   string  `json:"source_repo"`
	Source       string  `json:"source"`
	Target       string  `json:"target"`
	AuthorLogin  string  `json:"author_login"`
	AuthorName   string  `json:"author_name"`
	AuthorEmail  string  `json:"author_email"`
	AuthorAvatar string  `json:"author_avatar"`
	Sender       string  `json:"sender"`
	Started      int64   `json:"started"`
	Finished     int64   `json:"finished"`
	Created      int64   `json:"created"`
	Updated      int64   `json:"updated"`
	Version      int     `json:"version"`
	Stages       []Stage `json:"stages"`
}

type Stage struct {
	ID        int64  `json:"id"`
	RepoID    int64  `json:"repo_id"`
	BuildID   int64  `json:"build_id"`
	Number    int    `json:"number"`
	Name      string `json:"name"`
	Kind      string `json:"kind"`
	Type      string `json:"type"`
	Status    string `json:"status"`
	ErrIgnore bool   `json:"errignore"`
	ExitCode  int    `json:"exit_code"`
	Machine   string `json:"machine"`
	OS        string `json:"os"`
	Arch      string `json:"arch"`
	Started   int64  `json:"started"`
	Stopped   int64  `json:"stopped"`
	Created   int64  `json:"created"`
	Updated   int64  `json:"updated"`
	Version   int    `json:"version"`
	OnSuccess bool   `json:"on_success"`
	OnFailure bool   `json:"on_failure"`
}
