//go:build e2e

package e2e

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPipelineListPage(t *testing.T) {
	tests := []struct {
		name         string
		pipelineName string
		wantEmpty    bool
		wantRedirect bool
		wantInList   string
	}{
		{
			name:      "empty state shows no pipelines message",
			wantEmpty: true,
		},
		{
			name:         "create pipeline via modal redirects to editor",
			pipelineName: "e2e-pipeline-1",
			wantRedirect: true,
		},
		{
			name:         "create pipeline and verify in list",
			pipelineName: "e2e-pipeline-2",
			wantInList:   "e2e-pipeline-2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ResetDB(t)
			page := loginViaCookie(t)
			page.MustNavigate(URL("/service/web/pipelines"))
			wait := page.MustWaitRequestIdle()
			wait()

			if tt.pipelineName != "" {
				page.MustElement(`[data-testid="btn-new-pipeline"]`).MustClick()
				wait = page.MustWaitRequestIdle()
				wait()

				page.MustElement(`[data-testid="input-pipeline-name"]`).MustInput(tt.pipelineName)
				page.MustElement(`[data-testid="btn-submit-create"]`).MustClick()
				wait = page.MustWaitRequestIdle()
				wait()

				if tt.wantRedirect {
					info := page.MustInfo()
					assert.Contains(t, info.URL, "/service/web/pipelines/"+tt.pipelineName)
				}
				if tt.wantInList != "" {
					page.MustNavigate(URL("/service/web/pipelines"))
					wait = page.MustWaitRequestIdle()
					wait()
					body := page.MustElement("body").MustText()
					assert.Contains(t, body, tt.wantInList)
				}
			}

			if tt.wantEmpty {
				body := page.MustElement("body").MustText()
				assert.Contains(t, body, "No pipelines yet.")
			}
		})
	}
}

func TestPipelineDelete(t *testing.T) {
	tests := []struct {
		name             string
		pipelineName     string
		confirmDelete    bool
		shouldStillExist bool
	}{
		{
			name:             "delete removes pipeline from list",
			pipelineName:     "e2e-del-pipe",
			confirmDelete:    true,
			shouldStillExist: false,
		},
		{
			name:             "cancel delete keeps pipeline",
			pipelineName:     "e2e-keep-pipe",
			confirmDelete:    false,
			shouldStillExist: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ResetDB(t)

			seedPipeline(t, tt.pipelineName)

			page := loginViaCookie(t)
			page.MustNavigate(URL("/service/web/pipelines"))
			wait := page.MustWaitRequestIdle()
			wait()

			// Register dialog handler before clicking delete.
			// The delete button uses hx-confirm which triggers a native browser confirm.
			waitDialog, handleDialog := page.MustHandleDialog()
			go func() {
				waitDialog()
				handleDialog(tt.confirmDelete, "")
			}()

			page.MustElement(`[hx-confirm]`).MustClick()
			wait = page.MustWaitRequestIdle()
			wait()

			body := page.MustElement("body").MustText()
			if tt.shouldStillExist {
				assert.Contains(t, body, tt.pipelineName)
			} else {
				assert.NotContains(t, body, tt.pipelineName)
			}
		})
	}
}
