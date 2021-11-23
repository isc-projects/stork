package dump_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	dbmodel "isc.org/stork/server/database/model"
	"isc.org/stork/server/dumper/dump"
)

type logSourceMockCall struct {
	AgentAddress string
	AgentPort    int64
	Path         string
	Offset       int64
}

type logSourceMock struct {
	logs  []string
	err   error
	Calls []*logSourceMockCall
}

func newLogSourceMock(logs []string, err error) *logSourceMock {
	return &logSourceMock{logs, err, []*logSourceMockCall{}}
}

func (s *logSourceMock) TailTextFile(ctx context.Context, agentAddress string, agentPort int64, path string, offset int64) ([]string, error) {
	s.Calls = append(s.Calls, &logSourceMockCall{
		agentAddress,
		agentPort,
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
	require.EqualValues(t, &logSourceMockCall{
		"foo",
		42,
		"/var/log/foo.log",
		4000,
	}, logSource.Calls[0])
}
