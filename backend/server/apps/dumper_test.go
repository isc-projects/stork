package apps

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestCreateFileProducer(t *testing.T) {
	// Act
	fileProducer := prepareFileProducer("")

	// Assert
	require.NotNil(t, fileProducer)
}

func TestCreateFileUsingProducerWithExistingRoot(t *testing.T) {
	// Arrange
	root, _ := ioutil.TempDir("", "*")
	defer os.RemoveAll(root)
	fileProducer := prepareFileProducer(root)

	// Act
	file, fileErr := fileProducer("filename.ext")

	// Assert
	require.NoError(t, fileErr)
	file.Close()
}

func TestCreateFileUsingProducerWithExistingRootIsWritable(t *testing.T) {
	// Arrange
	root, _ := ioutil.TempDir("", "*")
	defer os.RemoveAll(root)
	fileProducer := prepareFileProducer(root)
	file, _ := fileProducer("filename.ext")
	defer file.Close()

	// Act
	_, err := file.WriteString("Hello World!")

	// Assert
	require.NoError(t, err)
}

func TestCreateFileProducerWithNonExistRoot(t *testing.T) {
	// Arrange
	root := "/non/exist/directory"
	fileProducer := prepareFileProducer(root)

	// Act
	file, err := fileProducer("filename.ext")

	// Assert
	require.Nil(t, file)
	require.Error(t, err)
}

func TestSaveToFileNoErrorOnProperInput(t *testing.T) {
	// Arrange
	root, _ := ioutil.TempDir("", "*")
	defer os.RemoveAll(root)
	fileProducer := prepareFileProducer(root)

	// Act
	err := saveToFile(fileProducer, "filename", []int{1, 2, 3, 4, 5})

	// Assert
	require.NoError(t, err)
}

func TestSaveToFileCreatesFile(t *testing.T) {
	// Arrange
	root, _ := ioutil.TempDir("", "*")
	defer os.RemoveAll(root)
	fileProducer := prepareFileProducer(root)

	// Act
	_ = saveToFile(fileProducer, "filename", []int{1, 2, 3, 4, 5})
	fileInfos, err := ioutil.ReadDir(root)

	// Assert
	require.NoError(t, err)
	require.Len(t, fileInfos, 1)
}

func TestSaveToFileCreatesFileWithProperFilename(t *testing.T) {
	// Arrange
	root, _ := ioutil.TempDir("", "*")
	defer os.RemoveAll(root)
	fileProducer := prepareFileProducer(root)

	// Act
	_ = saveToFile(fileProducer, "filename", []int{1, 2, 3, 4, 5})
	fileInfos, _ := ioutil.ReadDir(root)
	fileInfo := fileInfos[0]
	filename := fileInfo.Name()

	// Assert
	require.Contains(t, filename, "filename")
	require.True(t, strings.HasSuffix(filename, ".json"))
}

func TestSaveToFileCreatesFileWithNoEmptyContent(t *testing.T) {
	// Arrange
	root, _ := ioutil.TempDir("", "*")
	defer os.RemoveAll(root)
	fileProducer := prepareFileProducer(root)

	// Act
	_ = saveToFile(fileProducer, "filename", []int{1, 2, 3, 4, 5})
	fileInfos, _ := ioutil.ReadDir(root)
	fileInfo := fileInfos[0]
	filename := fileInfo.Name()
	filePath := path.Join(root, filename)
	content, err := ioutil.ReadFile(filePath)

	// Assert
	require.NoError(t, err)
	require.NotEmpty(t, content)
}

func TestSaveToFileCreatesFileWithProperContent(t *testing.T) {
	// Arrange
	root, _ := ioutil.TempDir("", "*")
	defer os.RemoveAll(root)
	fileProducer := prepareFileProducer(root)

	// Act
	_ = saveToFile(fileProducer, "filename", []int{1, 2, 3, 4, 5})
	fileInfos, _ := ioutil.ReadDir(root)
	fileInfo := fileInfos[0]
	filename := fileInfo.Name()
	filePath := path.Join(root, filename)
	content, _ := ioutil.ReadFile(filePath)
	var items []int
	err := json.Unmarshal(content, &items)

	// Assert
	require.NoError(t, err)
	require.EqualValues(t, []int{1, 2, 3, 4, 5}, items)
}

func TestSaveToFileWithNilContent(t *testing.T) {
	// Arrange
	root, _ := ioutil.TempDir("", "*")
	defer os.RemoveAll(root)
	fileProducer := prepareFileProducer(root)

	// Act
	err := saveToFile(fileProducer, "filename", nil)

	// Assert
	require.NoError(t, err)
}

func TestDumpSummaryToValidDirectory(t *testing.T) {
	// Arrange
	root, _ := ioutil.TempDir("", "*")
	defer os.RemoveAll(root)
	fileProducer := prepareFileProducer(root)
	content := []*dumpResult{}

	// Act
	dumpErr := saveDumpSummary(fileProducer, content)
	fileInfos, _ := ioutil.ReadDir(root)

	// Assert
	require.NoError(t, dumpErr)
	require.Len(t, fileInfos, 1)
}

func TestDumpSummaryToInalidDirectory(t *testing.T) {
	// Arrange
	root := "/not/exist/directory"
	fileProducer := prepareFileProducer(root)
	content := []*dumpResult{}

	// Act
	dumpErr := saveDumpSummary(fileProducer, content)

	// Assert
	require.Error(t, dumpErr)
}

func TestDumpSummaryWithNilContent(t *testing.T) {
	// Arrange
	root, _ := ioutil.TempDir("", "*")
	defer os.RemoveAll(root)
	fileProducer := prepareFileProducer(root)
	var content []*dumpResult = nil

	// Act
	dumpErr := saveDumpSummary(fileProducer, content)

	// Assert
	require.NoError(t, dumpErr)
}

func TestDumpSummaryFilename(t *testing.T) {
	// Arrange
	root, _ := ioutil.TempDir("", "*")
	defer os.RemoveAll(root)
	fileProducer := prepareFileProducer(root)
	content := []*dumpResult{}

	// Act
	_ = saveDumpSummary(fileProducer, content)
	fileInfos, _ := ioutil.ReadDir(root)
	fileInfo := fileInfos[0]
	filename := fileInfo.Name()
	filePath := path.Join(root, filename)

	// Assert
	require.Contains(t, filePath, "summary")
	require.True(t, strings.HasSuffix(filename, ".json"))
}

func TestDumpSummaryIsValidJSON(t *testing.T) {
	// Arrange
	root, _ := ioutil.TempDir("", "*")
	defer os.RemoveAll(root)
	fileProducer := prepareFileProducer(root)
	content := []*dumpResult{}

	// Act
	_ = saveDumpSummary(fileProducer, content)
	fileInfos, _ := ioutil.ReadDir(root)
	filePath := path.Join(root, fileInfos[0].Name())
	raw, _ := ioutil.ReadFile(filePath)
	var data dumpSummary
	err := json.Unmarshal(raw, &data)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, data)
	require.Len(t, data.Dumps, 0)
}

func TestDumpSummaryHasActualTimestamp(t *testing.T) {
	// Arrange
	root, _ := ioutil.TempDir("", "*")
	defer os.RemoveAll(root)
	fileProducer := prepareFileProducer(root)
	content := []*dumpResult{}

	// Act
	_ = saveDumpSummary(fileProducer, content)
	fileInfos, _ := ioutil.ReadDir(root)
	filePath := path.Join(root, fileInfos[0].Name())
	raw, _ := ioutil.ReadFile(filePath)
	var data dumpSummary
	_ = json.Unmarshal(raw, &data)

	timestampObj, err := time.Parse("2006-01-02T15:04:05 UTC", data.Timestamp)

	// Assert
	require.NoError(t, err)
	diff := time.Now().UTC().Sub(timestampObj)
	require.True(t, diff.Minutes() < 1.)
}

func TestDumpSummaryWithErrorResult(t *testing.T) {
	// Arrange
	root, _ := ioutil.TempDir("", "*")
	defer os.RemoveAll(root)
	fileProducer := prepareFileProducer(root)
	content := []*dumpResult{
		{
			DumpName: "foo",
			Error:    fmt.Errorf("bar"),
		},
	}

	require.NotNil(t, fileProducer)
	require.NotNil(t, content)
}

func TestDumpSummaryWithSuccessResult(t *testing.T) {

}

func TestDumpSummaryWithMultipleResults(t *testing.T) {

}
