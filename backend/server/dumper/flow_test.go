package dumper

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"isc.org/stork/server/agentcomm"
	agentcommtest "isc.org/stork/server/agentcomm/test"
	dbmodel "isc.org/stork/server/database/model"
	dbtest "isc.org/stork/server/database/test"
	"isc.org/stork/server/dumper/dumps"
	storktest "isc.org/stork/server/test"
	storkutil "isc.org/stork/util"
)

func TestNamingConventionForStructureDump(t *testing.T) {
	// Arrange
	artifact := dumps.NewBasicStructArtifact("bar", nil)
	dump := dumps.NewBasicDump("foo", artifact)

	// Act
	filename := flatStructureWithTimestampNamingConvention(dump, artifact)

	// Assert
	_, _, err := storkutil.ParseTimestampPrefix(filename)
	require.NoError(t, err)
	require.True(t, strings.HasSuffix(filename, ".json"))
	require.Contains(t, filename, dump.Name())
	require.Contains(t, filename, artifact.Name())
}

func TestNamingConventionForBinaryDump(t *testing.T) {
	// Arrange
	artifact := dumps.NewBasicBinaryArtifact("bar", nil)
	dump := dumps.NewBasicDump("foo", artifact)

	// Act
	filename := flatStructureWithTimestampNamingConvention(dump, artifact)

	// Assert
	_, _, err := storkutil.ParseTimestampPrefix(filename)
	require.NoError(t, err)
	require.False(t, strings.HasSuffix(filename, ".json"))
	require.Contains(t, filename, dump.Name())
	require.Contains(t, filename, artifact.Name())
}

// Test that the naming convention creates the filename without illegal characters.
func TestNamingConventionReturnsValidFilenames(t *testing.T) {
	// Arrange
	characters := "!@#$%^&*()_+{}:\"<>?~10-=[];',./πœę©ßß←↓↓→óþ¨~^´`ł…ə’ŋæðśążźć„”ńµ≤≥ ̣|\\"

	cases := []dumps.Dump{
		dumps.NewBasicDump("foo",
			dumps.NewBasicArtifact("bar"),
			dumps.NewBasicArtifact("BAZ"),
			dumps.NewBasicArtifact("42"),
		),
		dumps.NewBasicDump("123", dumps.NewBasicArtifact("foobar")),
	}

	for _, ch := range characters {
		str := string(ch)
		cases = append(cases, dumps.NewBasicDump(str, dumps.NewBasicArtifact(str)))
	}

	// Act
	filenames := make([]string, 0)
	for _, dump := range cases {
		for i := 0; i < dump.NumberOfArtifacts(); i++ {
			artifact := dump.GetArtifact(i)
			filename := flatStructureWithTimestampNamingConvention(dump, artifact)
			filenames = append(filenames, filename)
		}
	}

	// Assert
	for _, filename := range filenames {
		require.True(t, storkutil.IsValidFilename(filename), fmt.Sprintf("Wrong filename: %s", filename))
	}
}

// Test that the machine dump is properly created.
func TestDumpMachineReturnsNoErrorWhenMachineExists(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	m := &dbmodel.Machine{
		ID:         0,
		Address:    "localhost",
		AgentPort:  8080,
		Authorized: true,
	}
	_ = dbmodel.AddMachine(db, m)
	_ = dbmodel.InitializeSettings(db)

	agents := agentcommtest.NewFakeAgents(nil, nil)
	defer agents.Shutdown()

	// Act
	result, err := DumpMachine(db, agents, m.ID)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, result)

	result.Close()
}

// Test that the machine dump contains expected data.
func TestDumpMachineReturnsProperContent(t *testing.T) {
	// Arrange
	db, _, teardown := dbtest.SetupDatabaseTestCase(t)
	defer teardown()

	m := &dbmodel.Machine{
		ID:         0,
		Address:    "localhost",
		AgentPort:  8080,
		Authorized: true,
	}
	_ = dbmodel.AddMachine(db, m)
	_ = dbmodel.InitializeSettings(db)

	settings := agentcomm.AgentsSettings{}
	fec := &storktest.FakeEventCenter{}
	agents := agentcomm.NewConnectedAgents(&settings, fec, []byte{}, []byte{}, []byte{})
	defer agents.Shutdown()
	result, _ := DumpMachine(db, agents, m.ID)
	defer result.Close()

	// Act
	filenames, err := storkutil.ListFilesInTarball(result)

	// Assert
	require.NoError(t, err)
	require.Len(t, filenames, 4)
}
