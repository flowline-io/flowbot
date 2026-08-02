package server

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/flowline-io/flowbot/pkg/notify"
)

func TestWireNotifyStores_InjectsRecordsBackend(t *testing.T) {
	prev := notify.GetNotifyStore()
	t.Cleanup(func() { notify.SetNotifyRecords(prev) })

	notify.SetNotifyRecords(nil)
	require.Nil(t, notify.GetNotifyStore())

	WireNotifyStores()
	require.NotNil(t, notify.GetNotifyStore())
}
