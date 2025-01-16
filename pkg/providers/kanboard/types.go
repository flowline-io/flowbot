package kanboard

type EventResponse struct {
	EventName   string `json:"event_name"`
	EventData   any    `json:"event_data"`
	EventAuthor string `json:"event_author"`
}

type Data struct {
	Comment Comment `json:"comment"`
	Task    Task    `json:"task"`
}

type Task struct {
	ID                  int    `json:"id"`
	Title               string `json:"title"`
	Description         string `json:"description"`
	DateCreation        int    `json:"date_creation"`
	DateCompleted       any    `json:"date_completed"`
	DateDue             int    `json:"date_due"`
	ColorID             string `json:"color_id"`
	ProjectID           int    `json:"project_id"`
	ColumnID            int    `json:"column_id"`
	OwnerID             int    `json:"owner_id"`
	Position            int    `json:"position"`
	Score               int    `json:"score"`
	IsActive            int    `json:"is_active"`
	CategoryID          int    `json:"category_id"`
	CreatorID           int    `json:"creator_id"`
	DateModification    int    `json:"date_modification"`
	Reference           string `json:"reference"`
	DateStarted         int    `json:"date_started"`
	TimeSpent           int    `json:"time_spent"`
	TimeEstimated       int    `json:"time_estimated"`
	SwimlaneID          int    `json:"swimlane_id"`
	DateMoved           int    `json:"date_moved"`
	RecurrenceStatus    int    `json:"recurrence_status"`
	RecurrenceTrigger   int    `json:"recurrence_trigger"`
	RecurrenceFactor    int    `json:"recurrence_factor"`
	RecurrenceTimeframe int    `json:"recurrence_timeframe"`
	RecurrenceBasedate  int    `json:"recurrence_basedate"`
	RecurrenceParent    any    `json:"recurrence_parent"`
	RecurrenceChild     any    `json:"recurrence_child"`
	Priority            int    `json:"priority"`
	ExternalProvider    any    `json:"external_provider"`
	ExternalURI         any    `json:"external_uri"`
	CategoryName        any    `json:"category_name"`
	SwimlaneName        string `json:"swimlane_name"`
	ProjectName         string `json:"project_name"`
	ColumnTitle         string `json:"column_title"`
	AssigneeUsername    string `json:"assignee_username"`
	AssigneeName        any    `json:"assignee_name"`
	CreatorUsername     string `json:"creator_username"`
	CreatorName         any    `json:"creator_name"`
	CategoryDescription any    `json:"category_description"`
	ColumnPosition      int    `json:"column_position"`
}

type Comment struct {
	ID               int    `json:"id"`
	TaskID           int    `json:"task_id"`
	UserID           int    `json:"user_id"`
	DateCreation     int    `json:"date_creation"`
	DateModification int    `json:"date_modification"`
	Comment          string `json:"comment"`
	Reference        string `json:"reference"`
	Visibility       string `json:"visibility"`
	Username         string `json:"username"`
	Name             any    `json:"name"`
	Email            any    `json:"email"`
	AvatarPath       string `json:"avatar_path"`
}
