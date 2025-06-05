package agent

import (
	"testing"

	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"
)

// Test listing processes with eliminating child processes.
func TestListProcesses(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	proc1 := NewMockSupportedProcess(ctrl)
	proc1.EXPECT().getPid().AnyTimes().Return(int32(1))
	proc1.EXPECT().getName().AnyTimes().Return("kea-ctrl-agent", nil)
	proc1.EXPECT().getParentPid().AnyTimes().Return(int32(5), nil)

	proc2 := NewMockSupportedProcess(ctrl)
	proc2.EXPECT().getPid().AnyTimes().Return(int32(2))
	proc2.EXPECT().getName().AnyTimes().Return("kea-ctrl-agent", nil)
	proc2.EXPECT().getParentPid().AnyTimes().Return(int32(1), nil)

	proc3 := NewMockSupportedProcess(ctrl)
	proc3.EXPECT().getPid().AnyTimes().Return(int32(3))
	proc3.EXPECT().getName().AnyTimes().Return("kea-ctrl-agent", nil)
	proc3.EXPECT().getParentPid().AnyTimes().Return(int32(2), nil)

	proc4 := NewMockSupportedProcess(ctrl)
	proc4.EXPECT().getPid().AnyTimes().Return(int32(4))
	proc4.EXPECT().getName().AnyTimes().Return("kea-ctrl-agent", nil)
	proc4.EXPECT().getParentPid().AnyTimes().Return(int32(5), nil)

	lister := NewMockProcessLister(ctrl)
	lister.EXPECT().listProcesses().Return([]supportedProcess{proc1, proc2, proc3, proc4}, nil)

	pm := NewProcessManager()
	pm.lister = lister
	processes, err := pm.ListProcesses()
	require.NoError(t, err)
	require.Len(t, processes, 2)

	require.EqualValues(t, 1, processes[0].getPid())
	require.EqualValues(t, 4, processes[1].getPid())
}
