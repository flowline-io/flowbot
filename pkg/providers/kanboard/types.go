package kanboard

type EventResponse struct {
	EventName   string `json:"event_name,omitempty"`
	EventData   any    `json:"event_data,omitempty"`
	EventAuthor string `json:"event_author,omitempty"`
}

type Data struct {
	Comment Comment `json:"comment,omitempty"`
	Task    Task    `json:"task,omitempty"`
}

type Task struct {
	ID                  int    `json:"id,omitempty"`
	Title               string `json:"title"`
	Description         string `json:"description,omitempty"`
	DateCreation        int    `json:"date_creation,omitempty"`
	DateCompleted       any    `json:"date_completed,omitempty"`
	DateDue             int    `json:"date_due,omitempty"`
	ColorID             string `json:"color_id,omitempty"`
	ProjectID           int    `json:"project_id"`
	ColumnID            int    `json:"column_id,omitempty"`
	OwnerID             int    `json:"owner_id,omitempty"`
	Position            int    `json:"position,omitempty"`
	Score               int    `json:"score,omitempty"`
	IsActive            int    `json:"is_active,omitempty"`
	CategoryID          int    `json:"category_id,omitempty"`
	CreatorID           int    `json:"creator_id,omitempty"`
	DateModification    int    `json:"date_modification,omitempty"`
	Reference           string `json:"reference,omitempty"`
	DateStarted         int    `json:"date_started,omitempty"`
	TimeSpent           int    `json:"time_spent,omitempty"`
	TimeEstimated       int    `json:"time_estimated,omitempty"`
	SwimlaneID          int    `json:"swimlane_id,omitempty"`
	DateMoved           int    `json:"date_moved,omitempty"`
	RecurrenceStatus    int    `json:"recurrence_status,omitempty"`
	RecurrenceTrigger   int    `json:"recurrence_trigger,omitempty"`
	RecurrenceFactor    int    `json:"recurrence_factor,omitempty"`
	RecurrenceTimeframe int    `json:"recurrence_timeframe,omitempty"`
	RecurrenceBasedate  int    `json:"recurrence_basedate,omitempty"`
	RecurrenceParent    any    `json:"recurrence_parent,omitempty"`
	RecurrenceChild     any    `json:"recurrence_child,omitempty"`
	Priority            int    `json:"priority,omitempty"`
	ExternalProvider    any    `json:"external_provider,omitempty"`
	ExternalURI         any    `json:"external_uri,omitempty"`
	CategoryName        any    `json:"category_name,omitempty"`
	SwimlaneName        string `json:"swimlane_name,omitempty"`
	ProjectName         string `json:"project_name,omitempty"`
	ColumnTitle         string `json:"column_title,omitempty"`
	AssigneeUsername    string `json:"assignee_username,omitempty"`
	AssigneeName        any    `json:"assignee_name,omitempty"`
	CreatorUsername     string `json:"creator_username,omitempty"`
	CreatorName         any    `json:"creator_name,omitempty"`
	CategoryDescription any    `json:"category_description,omitempty"`
	ColumnPosition      int    `json:"column_position,omitempty"`
	Tags                []any  `json:"tags,omitempty"`
}

type Comment struct {
	ID               int    `json:"id,omitempty"`
	TaskID           int    `json:"task_id,omitempty"`
	UserID           int    `json:"user_id,omitempty"`
	DateCreation     int    `json:"date_creation,omitempty"`
	DateModification int    `json:"date_modification,omitempty"`
	Comment          string `json:"comment,omitempty"`
	Reference        string `json:"reference,omitempty"`
	Visibility       string `json:"visibility,omitempty"`
	Username         string `json:"username,omitempty"`
	Name             any    `json:"name,omitempty"`
	Email            any    `json:"email,omitempty"`
	AvatarPath       string `json:"avatar_path,omitempty"`
}
