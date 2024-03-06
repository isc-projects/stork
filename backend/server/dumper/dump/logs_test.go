package dump_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	dbmodel "isc.org/stork/server/database/model"
	"isc.org/stork/server/dumper/dump"
)

type logSourceMockCall struct {
	machine dbmodel.MachineTag
	Path    string
	Offset  int64
}

type logSourceMock struct {
	logs  []string
	err   error
	Calls []*logSourceMockCall
}

func newLogSourceMock(logs []string, err error) *logSourceMock {
	return &logSourceMock{logs, err, []*logSourceMockCall{}}
}

func (s *logSourceMock) TailTextFile(ctx context.Context, machine dbmodel.MachineTag, path string, offset int64) ([]string, error) {
	s.Calls = append(s.Calls, &logSourceMockCall{
		machine,
		path,
		offset,
	})

	return s.logs, s.err
}

// Test that the dump is executed properly.
func TestLogDumpExecute(t *testing.T) {
	// Arrange
	logSource := newLogSourceMock([]string{"foo", "bar"}, nil)
	m := dbmodel.Machine{
		Address:   "foo",
		AgentPort: 42,
		Apps: []*dbmodel.App{
			{
				Daemons: []*dbmodel.Daemon{
					{
						LogTargets: []*dbmodel.LogTarget{
							{
								Output: "/var/log/foo.log",
							},
							{
								Output: "stdout",
							},
						},
					},
				},
			},
		},
	}

	// Act
	dump := dump.NewLogsDump(&m, logSource)
	err := dump.Execute()

	// Assert
	require.NoError(t, err)
	require.Len(t, logSource.Calls, 1) // Stdout output ignored.
	require.Equal(t, "/var/log/foo.log", logSource.Calls[0].Path)
	require.EqualValues(t, 40000, logSource.Calls[0].Offset)
	require.Equal(t, "foo", logSource.Calls[0].machine.GetAddress())
	require.EqualValues(t, 42, logSource.Calls[0].machine.GetAgentPort())
}
