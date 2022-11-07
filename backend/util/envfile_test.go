package storkutil

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// Cleans the test environment variables.
func cleanTestEnvironmentVariables() {
	variables := []string{
		"TEST_STORK_KEY",
		"TEST_STORK_KEY1",
		"TEST_STORK_KEY2",
		"TEST_STORK_KEY3",
	}

	for _, variable := range variables {
		os.Unsetenv(variable)
	}
}

// Runs the test case. Executes additional steps before and after invoking the
// test.
func TestMain(m *testing.M) {
	cleanTestEnvironmentVariables()
	code := m.Run()
	cleanTestEnvironmentVariables()
	os.Exit(code)
}

// Test that the test environment variables are erased before executing test
// case.
func TestCleanEnvironmentVariables(t *testing.T) {
	// Act
	_, ok := os.LookupEnv("TEST_STORK_KEY")

	// Assert
	require.False(t, ok)
}

// Test that loading a missing environment file causes an error.
func TestLoadMissingEnvironmentFile(t *testing.T) {
	// Arrange & Act
	err := LoadEnvironmentFile("/not/existing/file")

	// Assert
	require.Error(t, err)
}

// Test that the single line environment file content is loaded properly.
func TestLoadSingleLineEnvironmentContent(t *testing.T) {
	// Arrange
	content := "TEST_STORK_KEY=VALUE"

	// Act
	err := loadEnvironmentEntries(strings.NewReader(content))

	// Assert
	require.NoError(t, err)
	require.EqualValues(t, "VALUE", os.Getenv("TEST_STORK_KEY"))
}

// Test that the multi-line environment file content is loaded properly.
func TestLoadMultiLineEnvironmentContent(t *testing.T) {
	// Arrange
	content := `TEST_STORK_KEY1=VALUE1
				TEST_STORK_KEY2=VALUE2
				TEST_STORK_KEY3=VALUE3`

	// Act
	err := loadEnvironmentEntries(strings.NewReader(content))

	// Assert
	require.NoError(t, err)
	require.EqualValues(t, "VALUE1", os.Getenv("TEST_STORK_KEY1"))
	require.EqualValues(t, "VALUE2", os.Getenv("TEST_STORK_KEY2"))
	require.EqualValues(t, "VALUE3", os.Getenv("TEST_STORK_KEY3"))
}

// Test that the duplicates in the content are overwritten properly.
func TestLoadEnvironmentContentWithDuplicates(t *testing.T) {
	// Arrange
	content := `TEST_STORK_KEY1=VALUE1
				TEST_STORK_KEY1=VALUE2
				TEST_STORK_KEY1=VALUE3`

	// Act
	err := loadEnvironmentEntries(strings.NewReader(content))

	// Assert
	require.NoError(t, err)
	require.EqualValues(t, "VALUE3", os.Getenv("TEST_STORK_KEY1"))
}

// Test that the empty value in the environment file content is loaded properly.
func TestLoadEnvironmentContentWithEmptyValue(t *testing.T) {
	// Arrange
	content := "TEST_STORK_KEY="

	// Act
	err := loadEnvironmentEntries(strings.NewReader(content))

	// Assert
	require.NoError(t, err)
	require.EqualValues(t, "", os.Getenv("TEST_STORK_KEY"))
}

// Test that the missing value separator in the environment file content
// causes an error.
func TestLoadEnvironmentContentWithoutSeparator(t *testing.T) {
	// Arrange
	content := "TEST_STORK_KEY/VALUE"

	// Act
	err := loadEnvironmentEntries(strings.NewReader(content))

	// Assert
	require.Error(t, err)
}

// Test that the invalid line index is included in the error message.
func TestLoadEnvironmentContentInvalidLineIndex(t *testing.T) {
	// Arrange
	content := `TEST_STORK_KEY1=VALUE1
				TEST_STORK_KEY2=VALUE2
				INVALID`

	// Act
	err := loadEnvironmentEntries(strings.NewReader(content))

	// Assert
	require.ErrorContains(t, err, "invalid line 3 of environment file")
}

// Test that the commented lines are skipped.
func TestLoadEnvironmentContentWithComments(t *testing.T) {
	// Arrange
	content := `# TEST_STORK_KEY1=VALUE1
				TEST_STORK_KEY2=VALUE2
				# INVALID`

	// Act
	err := loadEnvironmentEntries(strings.NewReader(content))

	// Assert
	require.NoError(t, err)
	require.EqualValues(t, "VALUE2", os.Getenv("TEST_STORK_KEY2"))
	_, ok := os.LookupEnv("TEST_STORK_KEY1")
	require.False(t, ok)
}

// Test that the trailing whitespaces are trimmed.
func TestLoadEnvironmentContentWithTrailingCharacters(t *testing.T) {
	// Arrange
	content := `  # TEST_STORK_KEY1=VALUE1  
				  TEST_STORK_KEY2=VALUE2   
				  # INVALID`

	// Act
	err := loadEnvironmentEntries(strings.NewReader(content))

	// Assert
	require.NoError(t, err)
	require.EqualValues(t, "VALUE2", os.Getenv("TEST_STORK_KEY2"))
	_, ok := os.LookupEnv("TEST_STORK_KEY1")
	require.False(t, ok)
}
