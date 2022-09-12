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
	"isc.org/stork/server/dumper/dump"
	storktest "isc.org/stork/server/test/dbmodel"
	"isc.org/stork/testutil"
	storkutil "isc.org/stork/util"
)

// Test that the convention produces expected name
// for a struct dump.
func TestNamingConventionForStructureDump(t *testing.T) {
	// Arrange
	artifact := dump.NewBasicStructArtifact("bar", nil)
	dump := dump.NewBasicDump("foo", artifact)

	// Act
	filename := flatStructureWithTimestampNamingConvention(dump, artifact)

	// Assert
	_, _, extension, err := testutil.ParseTimestampFilename(filename)
	require.NoError(t, err)
	require.EqualValues(t, extension, ".json")
	require.Contains(t, filename, dump.GetName())
	require.Contains(t, filename, artifact.GetName())
	require.True(t, strings.HasSuffix(filename, ".json"))
}

// Test that the convention produces expected name
// for a binary dump.
func TestNamingConventionForBinaryDump(t *testing.T) {
	// Arrange
	artifact := dump.NewBasicBinaryArtifact("bar", ".ext", nil)
	dump := dump.NewBasicDump("foo", artifact)

	// Act
	filename := flatStructureWithTimestampNamingConvention(dump, artifact)

	// Assert
	_, _, extension, err := testutil.ParseTimestampFilename(filename)
	require.NoError(t, err)
	require.EqualValues(t, extension, ".ext")
	require.Contains(t, filename, dump.GetName())
	require.Contains(t, filename, artifact.GetName())
	require.True(t, strings.HasSuffix(filename, artifact.GetExtension()))
}

// Test that the naming convention creates the filename without illegal characters.
func TestNamingConventionReturnsValidFilenames(t *testing.T) {
	// Arrange
	characters := "!@#$%^&*()_+{}:\"<>?~10-=[];',./πœę©ßß←↓↓→óþ¨~^´`ł…ə’ŋæðśążźć„”ńµ≤≥ ̣|\\"

	cases := []dump.Dump{
		dump.NewBasicDump("foo",
			dump.NewBasicArtifact("bar", ".ear"),
			dump.NewBasicArtifact("BAZ", ".EAZ"),
			dump.NewBasicArtifact("42", ".24"),
		),
		dump.NewBasicDump("123", dump.NewBasicArtifact("foobar", ".efb")),
	}

	for _, ch := range characters {
		str := string(ch)
		cases = append(cases, dump.NewBasicDump(str, dump.NewBasicArtifact(str, "."+str)))
	}

	// Act
	filenames := make([]string, 0)
	for _, dump := range cases {
		for i := 0; i < dump.GetArtifactsNumber(); i++ {
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
	_ = dbmodel.InitializeSettings(db, 0)

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
	_ = dbmodel.InitializeSettings(db, 0)

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

// Test that the JSON serializer does not escape characters problematic for HTML.
func TestIndentJSONSerializerNoEscape(t *testing.T) {
	jsonInput := `{
        "abc": "<a string with escaped characters>"
    }`
	jsonOutput, err := indentJSONSerializer(jsonInput)
	require.NoError(t, err)
	require.Contains(t, string(jsonOutput), "<a string with escaped characters>")
}
