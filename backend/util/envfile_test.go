package storkutil

import (
	"os"
	"path"
	"strings"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"isc.org/stork/testutil"
)

// Test that loading a missing environment file causes an error.
func TestLoadMissingEnvironmentFile(t *testing.T) {
	// Arrange & Act
	sb := testutil.NewSandbox()
	defer sb.Close()
	data, err := loadEnvironmentFile(path.Join(sb.BasePath, "not-exists.env"))

	// Assert
	require.Error(t, err)
	require.Nil(t, data)
}

// Test that the single line environment file content is loaded properly.
func TestLoadSingleLineEnvironmentContent(t *testing.T) {
	// Arrange
	content := "TEST_STORK_KEY=VALUE"

	// Act
	data, err := loadEnvironmentEntries(strings.NewReader(content))

	// Assert
	require.NoError(t, err)
	require.Len(t, data, 1)
	require.EqualValues(t, "TEST_STORK_KEY", data[0].key)
	require.EqualValues(t, "VALUE", data[0].value)
}

// Test that the multi-line environment file content is loaded properly.
func TestLoadMultiLineEnvironmentContent(t *testing.T) {
	// Arrange
	content := `TEST_STORK_KEY1=VALUE1
				TEST_STORK_KEY2=VALUE2
				TEST_STORK_KEY3=VALUE3`

	// Act
	data, err := loadEnvironmentEntries(strings.NewReader(content))

	// Assert
	require.NoError(t, err)
	require.Len(t, data, 3)
	require.EqualValues(t, "TEST_STORK_KEY1", data[0].key)
	require.EqualValues(t, "VALUE1", data[0].value)
	require.EqualValues(t, "TEST_STORK_KEY2", data[1].key)
	require.EqualValues(t, "VALUE2", data[1].value)
	require.EqualValues(t, "TEST_STORK_KEY3", data[2].key)
	require.EqualValues(t, "VALUE3", data[2].value)
}

// Test that the duplicates in the content are overwritten properly.
func TestLoadEnvironmentContentWithDuplicates(t *testing.T) {
	// Arrange
	content := `TEST_STORK_KEY1=VALUE1
				TEST_STORK_KEY1=VALUE2
				TEST_STORK_KEY1=VALUE3`

	// Act
	data, err := loadEnvironmentEntries(strings.NewReader(content))

	// Assert
	require.NoError(t, err)
	require.Len(t, data, 1)
	require.EqualValues(t, "TEST_STORK_KEY1", data[0].key)
	require.EqualValues(t, "VALUE3", data[0].value)
}

// Test that the empty value in the environment file content is loaded properly.
func TestLoadEnvironmentContentWithEmptyValue(t *testing.T) {
	// Arrange
	content := "TEST_STORK_KEY="

	// Act
	data, err := loadEnvironmentEntries(strings.NewReader(content))

	// Assert
	require.Len(t, data, 1)
	require.NoError(t, err)
	require.EqualValues(t, "TEST_STORK_KEY", data[0].key)
	require.EqualValues(t, "", data[0].value)
}

// Test that the missing value separator in the environment file content
// causes an error.
func TestLoadEnvironmentContentWithoutSeparator(t *testing.T) {
	// Arrange
	content := "TEST_STORK_KEY/VALUE"

	// Act
	data, err := loadEnvironmentEntries(strings.NewReader(content))

	// Assert
	require.Error(t, err)
	require.Nil(t, data)
}

// Test that the missing key in the environment file content causes an error.
func TestLoadEnvironmentContentWithoutKey(t *testing.T) {
	// Arrange
	content := "=VALUE"

	// Act
	data, err := loadEnvironmentEntries(strings.NewReader(content))

	// Assert
	require.Error(t, err)
	require.Nil(t, data)
}

// Test that the invalid line index is included in the error message.
func TestLoadEnvironmentContentInvalidLineIndex(t *testing.T) {
	// Arrange
	content := `TEST_STORK_KEY1=VALUE1
				TEST_STORK_KEY2=VALUE2
				INVALID`

	// Act
	data, err := loadEnvironmentEntries(strings.NewReader(content))

	// Assert
	require.ErrorContains(t, err, "invalid line 3 of environment file")
	require.Nil(t, data)
}

// Test that the commented lines are skipped.
func TestLoadEnvironmentContentWithComments(t *testing.T) {
	// Arrange
	content := `# TEST_STORK_KEY1=VALUE1
				TEST_STORK_KEY2=VALUE2
				# INVALID`

	// Act
	data, err := loadEnvironmentEntries(strings.NewReader(content))

	// Assert
	require.NoError(t, err)
	require.Len(t, data, 1)
	require.EqualValues(t, "TEST_STORK_KEY2", data[0].key)
	require.EqualValues(t, "VALUE2", data[0].value)
}

// Test that the empty lines are skipped.
func TestLoadEnvironmentContentWithEmptyLine(t *testing.T) {
	// Arrange
	content := `TEST_STORK_KEY1=VALUE1

				TEST_STORK_KEY2=VALUE2`

	// Act
	data, err := loadEnvironmentEntries(strings.NewReader(content))

	// Assert
	require.NoError(t, err)
	require.EqualValues(t, "TEST_STORK_KEY1", data[0].key)
	require.EqualValues(t, "VALUE1", data[0].value)
	require.EqualValues(t, "TEST_STORK_KEY2", data[1].key)
	require.EqualValues(t, "VALUE2", data[1].value)
	require.Len(t, data, 2)
}

// Test that the trailing whitespaces are trimmed.
func TestLoadEnvironmentContentWithTrailingCharacters(t *testing.T) {
	// Arrange
	content := `  # TEST_STORK_KEY1=VALUE1  
				  TEST_STORK_KEY2=VALUE2   
				  # INVALID`

	// Act
	data, err := loadEnvironmentEntries(strings.NewReader(content))

	// Assert
	require.NoError(t, err)
	require.Len(t, data, 1)
	require.EqualValues(t, "TEST_STORK_KEY2", data[0].key)
	require.EqualValues(t, "VALUE2", data[0].value)
}

type setterMock struct {
	data map[string]string
	err  error
}

func newEnvironmentVariableSetterMock(err error) *setterMock {
	return &setterMock{make(map[string]string), err}
}

func (s *setterMock) Set(key, value string) error {
	s.data[key] = value
	return s.err
}

// Test that the environment variables are loaded to the setter properly.
func TestLoadEnvironmentVariablesToSetter(t *testing.T) {
	// Arrange
	file, _ := os.CreateTemp("", "stork-envfile-test-*")
	defer (func() {
		file.Close()
		os.Remove(file.Name())
	})()

	content := `TEST_STORK_KEY=VALUE`
	_, _ = file.WriteString(content)

	mock := newEnvironmentVariableSetterMock(nil)

	// Act
	err := LoadEnvironmentFileToSetter(file.Name(), mock)

	// Assert
	require.NoError(t, err)
	require.EqualValues(t, "VALUE", mock.data["TEST_STORK_KEY"])
}

// Test that the loading error is returned properly if the setter object is used.
func TestLoadEnvironmentVariablesToSetterWithError(t *testing.T) {
	// Arrange
	file, _ := os.CreateTemp("", "stork-envfile-test-*")
	defer (func() {
		file.Close()
		os.Remove(file.Name())
	})()

	content := `TEST_STORK_KEY1=VALUE1
				TEST_STORK_KEY2=VALUE2`
	_, _ = file.WriteString(content)

	mock := newEnvironmentVariableSetterMock(errors.New("foo"))

	// Act
	err := LoadEnvironmentFileToSetter(file.Name(), mock)

	// Assert
	require.ErrorContains(t, err, "foo")
	require.NotContains(t, mock.data, "TEST_STORK_KEY2")
}

// Test that the process setter is constructed properly.
func TestNewProcessEnvironmentVariableSetter(t *testing.T) {
	// Act
	setter := NewProcessEnvironmentVariableSetter()

	// Assert
	require.NotNil(t, setter)
}

// Test that the process setter sets the key-value pair properly.
func TestProcessEnvironmentVariableSetterAcceptsValidPair(t *testing.T) {
	// Arrange
	t.Cleanup(testutil.CreateEnvironmentRestorePoint())
	setter := NewProcessEnvironmentVariableSetter()

	// Act
	err := setter.Set("TEST_STORK_KEY", "VALUE")

	// Assert
	require.NoError(t, err)
	_, ok := os.LookupEnv("TEST_STORK_KEY")
	require.True(t, ok)
}

// Test that process setter returns an error on invalid key-value pair.
func TestProcessEnvironmentVariableSetterRejectInvalidPair(t *testing.T) {
	// Arrange
	t.Cleanup(testutil.CreateEnvironmentRestorePoint())
	setter := NewProcessEnvironmentVariableSetter()

	// Act
	err := setter.Set("", "VALUE")

	// Assert
	require.Error(t, err)
}
