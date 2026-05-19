package model

import (
	"time"
)


// App mapped from table <apps>
type App struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Path        string    `json:"path"`
	ContainerID string    `json:"container_id"`
	Status      AppStatus `json:"status"`
	DockerInfo  JSON      `json:"docker_info"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
