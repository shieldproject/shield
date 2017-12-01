package shield

type Response struct {
	OK string `json:"ok"`

	TaskUUID string `json:"task_uuid,omitempty"`
}
