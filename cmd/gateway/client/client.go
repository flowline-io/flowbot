// Package client talks to Flowbot /gateway/v1 worker APIs.
package client

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/bytedance/sonic"
	"resty.dev/v3"

	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/protocol"
)

// Client talks to Flowbot /gateway/v1 APIs.
type Client struct {
	rc *resty.Client
}

// New builds a Flowbot gateway API client.
func New(baseURL, token string) *Client {
	rc := resty.New()
	rc.SetBaseURL(baseURL)
	rc.SetHeader("X-AccessToken", token)
	rc.SetHeader("Content-Type", "application/json")
	rc.SetTimeout(60 * time.Second)
	return &Client{rc: rc}
}

type apiCall struct {
	method string
	path   string
	body   any
}

func (c *Client) do(ctx context.Context, call apiCall) (protocol.Response, *resty.Response, error) {
	var resp protocol.Response
	req := c.rc.R().SetContext(ctx).SetResult(&resp)
	if call.body != nil {
		req.SetBody(call.body)
	}
	var httpResp *resty.Response
	var err error
	switch call.method {
	case http.MethodGet:
		httpResp, err = req.Get(call.path)
	case http.MethodPost:
		httpResp, err = req.Post(call.path)
	default:
		return resp, nil, fmt.Errorf("unsupported method %s", call.method)
	}
	if err != nil {
		return resp, nil, err
	}
	return resp, httpResp, nil
}

func decodeData[T any](data any) (T, error) {
	var out T
	raw, err := sonic.Marshal(data)
	if err != nil {
		return out, err
	}
	if err := sonic.Unmarshal(raw, &out); err != nil {
		return out, err
	}
	return out, nil
}

// Claim requests one pending job.
func (c *Client) Claim(ctx context.Context, workerID string) (*types.GatewayJob, error) {
	resp, httpResp, err := c.do(ctx, apiCall{
		method: http.MethodPost,
		path:   "/gateway/v1/claim",
		body:   types.GatewayClaimRequest{WorkerID: workerID},
	})
	if err != nil {
		return nil, err
	}
	if httpResp.StatusCode() >= 300 {
		return nil, fmt.Errorf("claim: status %d: %s", httpResp.StatusCode(), httpResp.String())
	}
	if resp.Status != protocol.Success {
		return nil, fmt.Errorf("claim failed: %s", resp.Message)
	}
	out, err := decodeData[types.GatewayClaimResponse](resp.Data)
	if err != nil {
		return nil, err
	}
	return out.Job, nil
}

// Complete reports a terminal job result.
func (c *Client) Complete(ctx context.Context, jobID string, req types.GatewayCompleteRequest) error {
	resp, httpResp, err := c.do(ctx, apiCall{
		method: http.MethodPost,
		path:   "/gateway/v1/jobs/" + jobID + "/result",
		body:   req,
	})
	if err != nil {
		return err
	}
	if httpResp.StatusCode() >= 300 {
		return fmt.Errorf("complete: status %d: %s", httpResp.StatusCode(), httpResp.String())
	}
	if resp.Status != protocol.Success {
		return fmt.Errorf("complete failed: %s", resp.Message)
	}
	return nil
}

// Heartbeat renews worker last-seen and optional job lease.
func (c *Client) Heartbeat(ctx context.Context, workerID, jobID string) error {
	_, httpResp, err := c.do(ctx, apiCall{
		method: http.MethodPost,
		path:   "/gateway/v1/heartbeat",
		body:   types.GatewayHeartbeatRequest{WorkerID: workerID, JobID: jobID},
	})
	if err != nil {
		return err
	}
	if httpResp.StatusCode() >= 300 {
		return fmt.Errorf("heartbeat: status %d: %s", httpResp.StatusCode(), httpResp.String())
	}
	return nil
}

// GetJob loads job status (used to detect cancel).
func (c *Client) GetJob(ctx context.Context, jobID string) (*types.GatewayJob, error) {
	resp, httpResp, err := c.do(ctx, apiCall{
		method: http.MethodGet,
		path:   "/gateway/v1/jobs/" + jobID,
	})
	if err != nil {
		return nil, err
	}
	if httpResp.StatusCode() == http.StatusNotFound {
		return nil, nil
	}
	if httpResp.StatusCode() >= 300 {
		return nil, fmt.Errorf("get job: status %d: %s", httpResp.StatusCode(), httpResp.String())
	}
	job, err := decodeData[types.GatewayJob](resp.Data)
	if err != nil {
		return nil, err
	}
	return &job, nil
}
