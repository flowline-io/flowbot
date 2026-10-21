package drone

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/capability"
	"github.com/flowline-io/flowbot/pkg/notify"
	droneprovider "github.com/flowline-io/flowbot/pkg/providers/drone"
	"github.com/flowline-io/flowbot/pkg/types"
)

var (
	errSend  = errors.New("send failed")
	errGitea = errors.New("gitea down")
)

type fakeBuilds struct {
	build *droneprovider.Build
	err   error
	ns    string
	repo  string
}

func (f *fakeBuilds) CreateBuild(namespace, name string) (*droneprovider.Build, error) {
	f.ns = namespace
	f.repo = name
	return f.build, f.err
}

func TestNewUnconfigured(t *testing.T) {
	t.Parallel()
	assert.Nil(t, New())
}

func TestNewWithDepsNil(t *testing.T) {
	t.Parallel()
	assert.Nil(t, NewWithDeps(nil, func(context.Context) (string, error) { return "alice", nil }, nil, nil))
	assert.Nil(t, NewWithDeps(&fakeBuilds{}, nil, nil, nil))
}

func TestDeploy(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		namespace namespaceFunc
		builds    *fakeBuilds
		notifyErr error
		droneURL  string
		uid       types.Uid
		wantErr   error
		wantNum   int
		wantNote  bool
	}{
		{
			name:      "creates build and notifies",
			namespace: func(context.Context) (string, error) { return "alice", nil },
			builds:    &fakeBuilds{build: &droneprovider.Build{Number: 42}},
			droneURL:  "https://drone.example",
			uid:       "user-1",
			wantNum:   42,
			wantNote:  true,
		},
		{
			name:      "omits url when drone base is empty",
			namespace: func(context.Context) (string, error) { return "bob", nil },
			builds:    &fakeBuilds{build: &droneprovider.Build{Number: 1}},
			uid:       "user-2",
			wantNum:   1,
			wantNote:  true,
		},
		{
			name:      "soft skips missing notify defaults",
			namespace: func(context.Context) (string, error) { return "carol", nil },
			builds:    &fakeBuilds{build: &droneprovider.Build{Number: 7}},
			notifyErr: notify.ErrNoDefaultTemplate,
			wantNum:   7,
			wantNote:  true,
		},
		{
			name:      "returns notify failure",
			namespace: func(context.Context) (string, error) { return "dave", nil },
			builds:    &fakeBuilds{build: &droneprovider.Build{Number: 3}},
			notifyErr: errSend,
			wantErr:   errSend,
			wantNote:  true,
		},
		{
			name:      "wraps create build failure",
			namespace: func(context.Context) (string, error) { return "erin", nil },
			builds:    &fakeBuilds{err: errors.New("http")},
			wantErr:   types.ErrProvider,
		},
		{
			name:      "rejects empty namespace",
			namespace: func(context.Context) (string, error) { return "", nil },
			builds:    &fakeBuilds{},
			wantErr:   types.ErrProvider,
		},
		{
			name:      "returns namespace error",
			namespace: func(context.Context) (string, error) { return "", errGitea },
			builds:    &fakeBuilds{},
			wantErr:   errGitea,
		},
		{
			name:      "rejects nil build",
			namespace: func(context.Context) (string, error) { return "frank", nil },
			builds:    &fakeBuilds{},
			wantErr:   types.ErrProvider,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var gotUID types.Uid
			var gotPayload map[string]any
			noted := false
			svc := NewWithDeps(tt.builds, tt.namespace, func(_ context.Context, uid types.Uid, payload map[string]any) error {
				noted = true
				gotUID = uid
				gotPayload = payload
				return tt.notifyErr
			}, func() string { return tt.droneURL })
			got, err := svc.Deploy(context.Background(), DeployInput{UID: tt.uid})
			if tt.wantErr != nil {
				require.Error(t, err)
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, tt.wantNum, got.Number)
			assert.Equal(t, droneprovider.DefaultDeployRepoName, got.Repo)
			assert.Equal(t, tt.droneURL, got.URL)
			assert.Equal(t, tt.wantNote, noted)
			if tt.wantNote && tt.notifyErr == nil {
				assert.Equal(t, tt.uid, gotUID)
				assert.Equal(t, deployNotifyPayload(got.Namespace, got.Repo, got.Number, tt.droneURL), gotPayload)
			}
			assert.Equal(t, droneprovider.DefaultDeployRepoName, tt.builds.repo)
		})
	}
}

func TestDeployCanceled(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	svc := NewWithDeps(&fakeBuilds{}, func(context.Context) (string, error) {
		t.Fatal("namespace should not be called")
		return "", nil
	}, nil, nil)
	_, err := svc.Deploy(ctx, DeployInput{})
	require.ErrorIs(t, err, types.ErrTimeout)
}

func TestHealthCheck(t *testing.T) {
	t.Parallel()
	svc := NewWithDeps(&fakeBuilds{}, func(context.Context) (string, error) { return "alice", nil }, nil, nil)
	ok, err := svc.HealthCheck(context.Background())
	require.NoError(t, err)
	assert.True(t, ok)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = svc.HealthCheck(ctx)
	require.ErrorIs(t, err, types.ErrTimeout)
}

func TestDeployNotifyPayload(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		user     string
		repo     string
		build    int
		droneURL string
		want     map[string]any
	}{
		{
			name:     "includes summary title and drone url",
			user:     "alice",
			repo:     "deploy",
			build:    42,
			droneURL: "https://drone.example",
			want: map[string]any{
				notify.PayloadKeyTitle:   "Deployment triggered",
				notify.PayloadKeySummary: "alice/deploy #42",
				notify.PayloadKeyURL:     "https://drone.example",
			},
		},
		{
			name:  "omits url when drone url is empty",
			user:  "bob",
			repo:  "deploy",
			build: 1,
			want: map[string]any{
				notify.PayloadKeyTitle:   "Deployment triggered",
				notify.PayloadKeySummary: "bob/deploy #1",
			},
		},
		{
			name:     "formats zero build number",
			user:     "carol",
			repo:     "deploy",
			build:    0,
			droneURL: "https://ci.local",
			want: map[string]any{
				notify.PayloadKeyTitle:   "Deployment triggered",
				notify.PayloadKeySummary: "carol/deploy #0",
				notify.PayloadKeyURL:     "https://ci.local",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := deployNotifyPayload(tt.user, tt.repo, tt.build, tt.droneURL)
			require.Equal(t, tt.want, got)
			_, hasURL := got[notify.PayloadKeyURL]
			assert.Equal(t, tt.droneURL != "", hasURL)
		})
	}
}

func TestInvokeDeploy(t *testing.T) {
	t.Parallel()
	svc := &captureService{build: &Build{Number: 9, Namespace: "alice", Repo: "deploy"}}
	res, err := invokeDeploy(svc, "drone")(context.Background(), map[string]any{"uid": "user-9"})
	require.NoError(t, err)
	require.Equal(t, types.Uid("user-9"), svc.in.UID)
	require.Equal(t, svc.build, res.Data)
	assert.Equal(t, "alice/deploy #9", res.Text)
	require.NotNil(t, res.Resource)
	assert.Equal(t, "9", res.Resource.EntityID)
	assert.Equal(t, "drone", res.Resource.App)
}

type captureService struct {
	in    DeployInput
	build *Build
	err   error
}

func (s *captureService) Deploy(_ context.Context, in DeployInput) (*Build, error) {
	s.in = in
	return s.build, s.err
}

func (*captureService) HealthCheck(context.Context) (bool, error) { return true, nil }

func TestInvokeDeployNilBuild(t *testing.T) {
	t.Parallel()
	_, err := invokeDeploy(&captureService{}, "drone")(context.Background(), nil)
	require.ErrorIs(t, err, types.ErrProvider)
}

var _ capability.Invoker = invokeDeploy(&captureService{}, "drone")
