package oneai

type Response struct {
	InputText interface{} `json:"input_text"`
	Input     []struct {
		Utterance string `json:"utterance"`
	} `json:"input"`
	Status string      `json:"status"`
	Error  interface{} `json:"error"`
	Output []struct {
		TextGeneratedByStepName string      `json:"text_generated_by_step_name"`
		TextGeneratedByStepID   int         `json:"text_generated_by_step_id"`
		Text                    interface{} `json:"text"`
		Contents                []struct {
			Utterance string `json:"utterance"`
		} `json:"contents"`
	} `json:"output"`
	Warnings interface{} `json:"warnings"`
	Stats    struct {
		ConcurrencyWaitTime interface{} `json:"concurrency_wait_time"`
		TotalRunningJobs    int         `json:"total_running_jobs"`
		TotalWaitingJobs    int         `json:"total_waiting_jobs"`
	} `json:"stats"`
}
