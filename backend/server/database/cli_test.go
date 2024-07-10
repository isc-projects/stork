package dbops

import (
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"isc.org/stork/testutil"
)

// Test that the field tags are handled properly.
func TestSetFieldsBasedOnTags(t *testing.T) {
	// Arrange
	type parentMock struct {
		FieldString string `tag:"nested-field-string"`
	}

	type mock struct {
		Parent                   parentMock
		FieldString              string `tag:"field-string"`
		FieldInt                 int    `tag:"field-int"`
		FieldWithoutTag          string
		FieldWithUnexpectedTag   string `unexpected:"tag"`
		FieldWithMultipleTags    string `tag:"field-multiple" another:"unexpected"`
		FieldWithUnsupportedType bool   `tag:"field-boolean"`
		FieldStringUnknown       string `tag:"field-unknown"`
	}

	lookup := func(key string) (string, bool) {
		switch key {
		case "field-string":
			return "value-string", true
		case "field-int":
			return "42", true
		case "field-multiple":
			return "value-multiple", true
		case "field-boolean":
			return "true", true
		case "nested-field-string":
			return "nested-field-string", true
		default:
			return "", false
		}
	}

	obj := &mock{}

	// Act
	setFieldsBasedOnTags(obj, "tag", lookup)

	// Assert
	require.EqualValues(t, "value-string", obj.FieldString)
	require.EqualValues(t, 42, obj.FieldInt)
	require.Empty(t, obj.FieldWithoutTag)
	require.Empty(t, obj.FieldWithUnexpectedTag)
	require.EqualValues(t, "value-multiple", obj.FieldWithMultipleTags)
	require.False(t, obj.FieldWithUnsupportedType)
	require.Empty(t, obj.FieldStringUnknown)
	require.EqualValues(t, "nested-field-string", obj.Parent.FieldString)
}

// Test that the values of the struct members are read from environment
// variables correctly.
func TestReadFromEnvironment(t *testing.T) {
	// Arrange
	restore := testutil.CreateEnvironmentRestorePoint()
	defer restore()

	type mock struct {
		String  string `env:"STRING"`
		Int     int    `env:"INT"`
		Bool    bool   `env:"BOOL"`
		Missing string `env:"MISSING"`
		NoTag   string
	}

	os.Setenv("STRING", "string")
	os.Setenv("INT", "42")
	os.Setenv("BOOL", "true")

	obj := &mock{}

	// Act
	readFromEnvironment(obj)

	// Assert
	require.EqualValues(t, "string", obj.String)
	require.EqualValues(t, 42, obj.Int)
	// Boolean is not supported.
	require.False(t, obj.Bool)
	require.Empty(t, obj.Missing)
	require.Empty(t, obj.NoTag)
}

// Test that the flags are read from the environment variables properly.
func TestReadDatabaseCLIFlagsFromEnvironment(t *testing.T) {
	// Arrange
	restore := testutil.CreateEnvironmentRestorePoint()
	defer restore()

	os.Setenv("STORK_DATABASE_NAME", "dbname")
	os.Setenv("STORK_DATABASE_USER_NAME", "user")
	os.Setenv("STORK_DATABASE_PASSWORD", "password")
	os.Setenv("STORK_DATABASE_HOST", "host")
	os.Setenv("STORK_DATABASE_PORT", "42")
	os.Setenv("STORK_DATABASE_SSLMODE", "sslmode")
	os.Setenv("STORK_DATABASE_SSLKEY", "sslkey")
	os.Setenv("STORK_DATABASE_READ_TIMEOUT", "24m")
	os.Setenv("STORK_DATABASE_WRITE_TIMEOUT", "42s")

	obj := &DatabaseCLIFlags{}

	// Act
	obj.ReadFromEnvironment()

	// Assert
	require.EqualValues(t, "dbname", obj.DBName)
	require.EqualValues(t, "user", obj.User)
	require.EqualValues(t, "password", obj.Password)
	require.EqualValues(t, "host", obj.Host)
	require.EqualValues(t, 42, obj.Port)
	require.EqualValues(t, "sslmode", obj.SSLMode)
	require.EqualValues(t, "sslkey", obj.SSLKey)
	require.EqualValues(t, 24*time.Minute, obj.ReadTimeout)
	require.EqualValues(t, 42*time.Second, obj.WriteTimeout)
}

// Test that the maintenance flags are read from the environment variables properly.
func TestReadMaintenanceDatabaseCLIFlagsFromEnvironment(t *testing.T) {
	// Arrange
	restore := testutil.CreateEnvironmentRestorePoint()
	defer restore()

	os.Setenv("STORK_DATABASE_MAINTENANCE_NAME", "maintenance-dbname")
	os.Setenv("STORK_DATABASE_MAINTENANCE_USER_NAME", "maintenance-user")
	os.Setenv("STORK_DATABASE_MAINTENANCE_PASSWORD", "maintenance-password")
	os.Setenv("STORK_DATABASE_NAME", "dbname")
	os.Setenv("STORK_DATABASE_USER_NAME", "user")
	os.Setenv("STORK_DATABASE_PASSWORD", "password")
	os.Setenv("STORK_DATABASE_HOST", "host")
	os.Setenv("STORK_DATABASE_PORT", "42")
	os.Setenv("STORK_DATABASE_SSLMODE", "sslmode")
	os.Setenv("STORK_DATABASE_SSLKEY", "sslkey")
	os.Setenv("STORK_DATABASE_READ_TIMEOUT", "24m")
	os.Setenv("STORK_DATABASE_WRITE_TIMEOUT", "42s")

	obj := &DatabaseCLIFlagsWithMaintenance{}

	// Act
	obj.ReadFromEnvironment()

	// Assert
	require.EqualValues(t, "dbname", obj.DBName)
	require.EqualValues(t, "user", obj.User)
	require.EqualValues(t, "password", obj.Password)
	require.EqualValues(t, "host", obj.Host)
	require.EqualValues(t, 42, obj.Port)
	require.EqualValues(t, "sslmode", obj.SSLMode)
	require.EqualValues(t, "sslkey", obj.SSLKey)
	require.EqualValues(t, 24*time.Minute, obj.ReadTimeout)
	require.EqualValues(t, 42*time.Second, obj.WriteTimeout)

	require.EqualValues(t, "maintenance-dbname", obj.MaintenanceDBName)
	require.EqualValues(t, "maintenance-user", obj.MaintenanceUser)
	require.EqualValues(t, "maintenance-password", obj.MaintenancePassword)
}

type mockCLILookup struct {
	values   map[string]string
	defaults map[string]bool
}

func newMockCLILookup(values map[string]string, defaults ...string) *mockCLILookup {
	defaultsMap := make(map[string]bool, len(defaults))
	for _, key := range defaults {
		defaultsMap[key] = true
	}

	return &mockCLILookup{values: values}
}

func (m *mockCLILookup) IsSet(key string) bool {
	_, hasValue := m.values[key]
	_, isDefault := m.defaults[key]
	return hasValue && !isDefault
}

func (m *mockCLILookup) String(key string) string {
	if value, ok := m.values[key]; ok {
		return value
	}
	return ""
}

// Test that the values of the struct members are read from CLI flags correctly.
func TestReadFromCLI(t *testing.T) {
	// Arrange
	type mock struct {
		FieldExisting string `long:"field-existing"`
		FieldMissing  string `long:"field-missing"`
		FieldDefault  string `long:"field-default"`
	}

	lookup := newMockCLILookup(map[string]string{
		"field-existing": "value-existing",
		"field-default":  "value-default",
	}, "field-default")

	obj := &mock{}

	// Act
	readFromCLI(obj, lookup)

	// Assert
	require.EqualValues(t, "value-existing", obj.FieldExisting)
	require.Empty(t, obj.FieldMissing)
	require.EqualValues(t, "value-default", obj.FieldDefault)
}

// Test that the database CLI flags are converted to the database settings
// properly.
func TestConvertDatabaseCLIFlagsToSettings(t *testing.T) {
	// Arrange
	cliFlags := &DatabaseCLIFlags{
		DBName:       "dbname",
		User:         "user",
		Password:     "password",
		Host:         "host",
		Port:         42,
		SSLMode:      "sslmode",
		SSLCert:      "sslcert",
		SSLKey:       "sslkey",
		SSLRootCert:  "sslrootcert",
		TraceSQL:     "run",
		ReadTimeout:  24 * time.Minute,
		WriteTimeout: 42 * time.Second,
	}

	// Act
	settings, err := cliFlags.ConvertToDatabaseSettings()

	// Assert
	require.NoError(t, err)
	require.EqualValues(t, "dbname", settings.DBName)
	require.EqualValues(t, "user", settings.User)
	require.EqualValues(t, "password", settings.Password)
	require.EqualValues(t, "host", settings.Host)
	require.EqualValues(t, 42, settings.Port)
	require.EqualValues(t, "sslmode", settings.SSLMode)
	require.EqualValues(t, "sslcert", settings.SSLCert)
	require.EqualValues(t, "sslkey", settings.SSLKey)
	require.EqualValues(t, "sslrootcert", settings.SSLRootCert)
	require.EqualValues(t, LoggingQueryPresetRuntime, settings.TraceSQL)
	require.EqualValues(t, 24*time.Minute, settings.ReadTimeout)
	require.EqualValues(t, 42*time.Second, settings.WriteTimeout)
}

// Test that the database CLI flags with URL are converted to the database
// settings properly.
func TestConvertDatabaseCLIFlagsWithURLToSettings(t *testing.T) {
	// Arrange
	cliFlags := &DatabaseCLIFlags{
		URL:          "postgres://user:password@host:42/dbname",
		SSLMode:      "sslmode",
		SSLCert:      "sslcert",
		SSLKey:       "sslkey",
		SSLRootCert:  "sslrootcert",
		TraceSQL:     "run",
		ReadTimeout:  24 * time.Minute,
		WriteTimeout: 42 * time.Second,
	}

	// Act
	settings, err := cliFlags.ConvertToDatabaseSettings()

	// Assert
	require.NoError(t, err)
	require.EqualValues(t, "dbname", settings.DBName)
	require.EqualValues(t, "user", settings.User)
	require.EqualValues(t, "password", settings.Password)
	require.EqualValues(t, "host", settings.Host)
	require.EqualValues(t, 42, settings.Port)
	require.EqualValues(t, "sslmode", settings.SSLMode)
	require.EqualValues(t, "sslcert", settings.SSLCert)
	require.EqualValues(t, "sslkey", settings.SSLKey)
	require.EqualValues(t, "sslrootcert", settings.SSLRootCert)
	require.EqualValues(t, LoggingQueryPresetRuntime, settings.TraceSQL)
	require.EqualValues(t, 24*time.Minute, settings.ReadTimeout)
	require.EqualValues(t, 42*time.Second, settings.WriteTimeout)
}

// Test that the database CLI flags cannot be converted to settings if they
// contain mutually exclusive parameters.
func TestConvertDatabaseCLIFlagsToSettingsWithMutuallyExclusives(t *testing.T) {
	// Arrange
	testLabels := []string{"dbname", "user", "password", "host", "port"}
	testCases := []*DatabaseCLIFlags{
		{
			URL:    "postgres://user:password@host:42/dbname",
			DBName: "dbname",
		},
		{
			URL:  "postgres://user:password@host:42/dbname",
			User: "user",
		},
		{
			URL:      "postgres://user:password@host:42/dbname",
			Password: "password",
		},
		{
			URL:  "postgres://user:password@host:42/dbname",
			Host: "host",
		},
		{
			URL:  "postgres://user:password@host:42/dbname",
			Port: 42,
		},
	}

	for i, flags := range testCases {
		t.Run(testLabels[i], func(t *testing.T) {
			// Act
			settings, err := flags.ConvertToDatabaseSettings()

			// Assert
			require.Nil(t, settings)
			require.Error(t, err)
		})
	}
}

// Test that the invalid URL cannot be used to create the database settings.
func TestConvertInvalidURLToDatabaseSettings(t *testing.T) {
	// Arrange
	flags := &DatabaseCLIFlags{
		URL: "foo://bar",
	}

	// Act
	settings, err := flags.ConvertToDatabaseSettings()

	// Assert
	require.Error(t, err)
	require.Nil(t, settings)
}

// Test that the CLI flags can be read from an external parameters source using
// the CLI lookup.
func TestReadDatabaseCLIFlagsFromCLILookup(t *testing.T) {
	// Arrange
	cliFlags := &DatabaseCLIFlags{}
	lookup := newMockCLILookup(map[string]string{
		"db-name":          "dbname",
		"db-user":          "user",
		"db-host":          "host",
		"db-port":          "42",
		"db-sslmode":       "sslmode",
		"db-sslkey":        "sslkey",
		"db-sslcert":       "sslcert",
		"db-sslrootcert":   "sslrootcert",
		"db-trace-queries": "run",
		"db-read-timeout":  "24m",
		"db-write-timeout": "42s",
	})

	// Act
	cliFlags.ReadFromCLI(lookup)

	// Assert
	require.EqualValues(t, "dbname", cliFlags.DBName)
	require.EqualValues(t, "user", cliFlags.User)
	require.Empty(t, cliFlags.Password)
	require.EqualValues(t, "host", cliFlags.Host)
	require.EqualValues(t, 42, cliFlags.Port)
	require.EqualValues(t, "sslmode", cliFlags.SSLMode)
	require.EqualValues(t, "sslcert", cliFlags.SSLCert)
	require.EqualValues(t, "sslkey", cliFlags.SSLKey)
	require.EqualValues(t, "sslrootcert", cliFlags.SSLRootCert)
	require.EqualValues(t, LoggingQueryPresetRuntime, cliFlags.TraceSQL)
	require.EqualValues(t, 24*time.Minute, cliFlags.ReadTimeout)
	require.EqualValues(t, 42*time.Second, cliFlags.WriteTimeout)
}

// Test that the CLI flags that contains the maintenance credentials are
// converted to the standard database settings properly.
func TestConvertDatabaseCLIFlagsWithMaintenanceCredentialsToSettings(t *testing.T) {
	// Arrange
	cliFlags := &DatabaseCLIFlagsWithMaintenance{
		DatabaseCLIFlags: DatabaseCLIFlags{
			DBName:       "dbname",
			User:         "user",
			Password:     "password",
			Host:         "host",
			Port:         42,
			SSLMode:      "sslmode",
			SSLCert:      "sslcert",
			SSLKey:       "sslkey",
			SSLRootCert:  "sslrootcert",
			TraceSQL:     "run",
			ReadTimeout:  24 * time.Minute,
			WriteTimeout: 42 * time.Second,
		},
		MaintenanceDBName:   "maintenance-dbname",
		MaintenanceUser:     "maintenance-user",
		MaintenancePassword: "maintenance-password",
	}

	// Act
	settings, err := cliFlags.ConvertToDatabaseSettings()

	// Assert
	require.NoError(t, err)
	require.EqualValues(t, "dbname", settings.DBName)
	require.EqualValues(t, "user", settings.User)
	require.EqualValues(t, "password", settings.Password)
	require.EqualValues(t, "host", settings.Host)
	require.EqualValues(t, 42, settings.Port)
	require.EqualValues(t, "sslmode", settings.SSLMode)
	require.EqualValues(t, "sslcert", settings.SSLCert)
	require.EqualValues(t, "sslkey", settings.SSLKey)
	require.EqualValues(t, "sslrootcert", settings.SSLRootCert)
	require.EqualValues(t, LoggingQueryPresetRuntime, settings.TraceSQL)
	require.EqualValues(t, 24*time.Minute, settings.ReadTimeout)
	require.EqualValues(t, 42*time.Second, settings.WriteTimeout)
}

// Test that the CLI flags that contains the maintenance credentials are
// converted to the maintenance database settings properly.
func TestConvertDatabaseCLIFlagsWithMaintenanceCredentialsToMaintenanceSettings(t *testing.T) {
	// Arrange
	cliFlags := &DatabaseCLIFlagsWithMaintenance{
		DatabaseCLIFlags: DatabaseCLIFlags{
			DBName:       "dbname",
			User:         "user",
			Password:     "password",
			Host:         "host",
			Port:         42,
			SSLMode:      "sslmode",
			SSLCert:      "sslcert",
			SSLKey:       "sslkey",
			SSLRootCert:  "sslrootcert",
			TraceSQL:     "run",
			ReadTimeout:  24 * time.Minute,
			WriteTimeout: 42 * time.Second,
		},
		MaintenanceDBName:   "maintenance-dbname",
		MaintenanceUser:     "maintenance-user",
		MaintenancePassword: "maintenance-password",
	}

	// Act
	settings, err := cliFlags.ConvertToMaintenanceDatabaseSettings()

	// Assert
	require.NoError(t, err)
	require.EqualValues(t, "maintenance-dbname", settings.DBName)
	require.EqualValues(t, "maintenance-user", settings.User)
	require.EqualValues(t, "maintenance-password", settings.Password)
	require.EqualValues(t, "host", settings.Host)
	require.EqualValues(t, 42, settings.Port)
	require.EqualValues(t, "sslmode", settings.SSLMode)
	require.EqualValues(t, "sslcert", settings.SSLCert)
	require.EqualValues(t, "sslkey", settings.SSLKey)
	require.EqualValues(t, "sslrootcert", settings.SSLRootCert)
	require.EqualValues(t, LoggingQueryPresetRuntime, settings.TraceSQL)
	require.EqualValues(t, 24*time.Minute, settings.ReadTimeout)
	require.EqualValues(t, 42*time.Second, settings.WriteTimeout)
}

// Test that the CLI flags can be read from an external parameters source using
// the CLI lookup.
func TestReadDatabaseCLIFlagsWithMaintenanceCredentialsFromCLILookup(t *testing.T) {
	// Arrange
	cliFlags := &DatabaseCLIFlagsWithMaintenance{}
	lookup := newMockCLILookup(map[string]string{
		"db-name":             "dbname",
		"db-user":             "user",
		"db-host":             "host",
		"db-port":             "42",
		"db-sslmode":          "sslmode",
		"db-sslkey":           "sslkey",
		"db-sslcert":          "sslcert",
		"db-sslrootcert":      "sslrootcert",
		"db-trace-queries":    "run",
		"db-maintenance-name": "maintenance-dbname",
		"db-maintenance-user": "maintenance-user",
		"db-read-timeout":     "24m",
		"db-write-timeout":    "42s",
	})

	// Act
	cliFlags.ReadFromCLI(lookup)

	// Assert
	require.EqualValues(t, "dbname", cliFlags.DBName)
	require.EqualValues(t, "user", cliFlags.User)
	require.Empty(t, cliFlags.Password)
	require.EqualValues(t, "host", cliFlags.Host)
	require.EqualValues(t, 42, cliFlags.Port)
	require.EqualValues(t, "sslmode", cliFlags.SSLMode)
	require.EqualValues(t, "sslcert", cliFlags.SSLCert)
	require.EqualValues(t, "sslkey", cliFlags.SSLKey)
	require.EqualValues(t, "sslrootcert", cliFlags.SSLRootCert)
	require.EqualValues(t, LoggingQueryPresetRuntime, cliFlags.TraceSQL)
	require.EqualValues(t, "maintenance-dbname", cliFlags.MaintenanceDBName)
	require.EqualValues(t, "maintenance-user", cliFlags.MaintenanceUser)
	require.Empty(t, cliFlags.MaintenancePassword)
	require.EqualValues(t, 24*time.Minute, cliFlags.ReadTimeout)
	require.EqualValues(t, 42*time.Second, cliFlags.WriteTimeout)
}

// Test that the field iteration is performed properly.
func TestIterateOverFields(t *testing.T) {
	// Arrange
	type nestedMock struct {
		Nested string
	}

	type mock struct {
		Parent nestedMock
		Foo    string
		Bar    int
		Baz    time.Duration
	}

	obj := &mock{
		Parent: nestedMock{
			Nested: "Nested",
		},
		Foo: "Foo",
		Bar: 42,
		Baz: 42 * time.Second,
	}
	numFields := 0

	// Act
	iterateOverFields(obj, func(field reflect.StructField, value reflect.Value) {
		// Assert
		switch field.Name {
		case "Nested":
			require.EqualValues(t, "Nested", value.String())
		case "Foo":
			require.EqualValues(t, "Foo", value.String())
		case "Bar":
			require.EqualValues(t, 42, value.Int())
		case "Baz":
			require.EqualValues(t, 42*time.Second, value.Interface())
		}
		numFields++
	})

	require.EqualValues(t, 4, numFields)
}

// Test that the field iteration is performed properly even if the object is nil.
func TestIterateOverFieldsOfNil(t *testing.T) {
	// Arrange
	type nestedMock struct {
		Nested string
	}

	type mock struct {
		Parent nestedMock
		Foo    string
		Bar    string
	}

	obj := (*mock)(nil)
	numFields := 0

	// Act
	iterateOverFields(obj, func(field reflect.StructField, value reflect.Value) {
		// Assert
		require.NotNil(t, field)
		require.False(t, value.IsValid())
		numFields++
	})

	require.EqualValues(t, 3, numFields)
}

// Test that the CLI flag definitions are read properly from the struct tags.
func TestConvertToCLIFlagDefinitions(t *testing.T) {
	// Arrange
	type mock struct {
		Foo                string `short:"f" long:"foo" description:"foofoo" env:"FOO" default:"foofoofoo"`
		MissingShort       string `long:"bar" description:"barbar" env:"BAR" default:"barbarbar"`
		MissingLong        string `short:"z" description:"bazbaz" env:"BAZ" default:"bazbazbaz"`
		MissingDescription string `short:"i" long:"biz" env:"BIZ" default:"bizbizbiz"`
		MissingEnv         string `short:"o" long:"boz" description:"bozboz" default:"bozbozboz"`
		MissingDefault     string `short:"u" long:"buz" description:"buzbuz" env:"BUZ"`
		MissingTags        string
	}

	obj := &mock{}

	// Act
	flags := convertToCLIFlagDefinitions(obj)

	// Assert
	require.Len(t, flags, 7)

	require.EqualValues(t, "f", flags[0].Short)
	require.EqualValues(t, "foo", flags[0].Long)
	require.EqualValues(t, "foofoo", flags[0].Description)
	require.EqualValues(t, "FOO", flags[0].EnvironmentVariable)
	require.EqualValues(t, "foofoofoo", flags[0].Default)

	require.Empty(t, flags[1].Short)
	require.EqualValues(t, "bar", flags[1].Long)
	require.EqualValues(t, "barbar", flags[1].Description)
	require.EqualValues(t, "BAR", flags[1].EnvironmentVariable)
	require.EqualValues(t, "barbarbar", flags[1].Default)

	require.EqualValues(t, "z", flags[2].Short)
	require.Empty(t, flags[2].Long)
	require.EqualValues(t, "bazbaz", flags[2].Description)
	require.EqualValues(t, "BAZ", flags[2].EnvironmentVariable)
	require.EqualValues(t, "bazbazbaz", flags[2].Default)

	require.EqualValues(t, "i", flags[3].Short)
	require.EqualValues(t, "biz", flags[3].Long)
	require.Empty(t, flags[3].Description)
	require.EqualValues(t, "BIZ", flags[3].EnvironmentVariable)
	require.EqualValues(t, "bizbizbiz", flags[3].Default)

	require.EqualValues(t, "o", flags[4].Short)
	require.EqualValues(t, "boz", flags[4].Long)
	require.EqualValues(t, "bozboz", flags[4].Description)
	require.Empty(t, flags[4].EnvironmentVariable)
	require.EqualValues(t, "bozbozboz", flags[4].Default)

	require.EqualValues(t, "u", flags[5].Short)
	require.EqualValues(t, "buz", flags[5].Long)
	require.EqualValues(t, "buzbuz", flags[5].Description)
	require.EqualValues(t, "BUZ", flags[5].EnvironmentVariable)
	require.Empty(t, flags[5].Default)

	require.Empty(t, flags[6].Short)
	require.Empty(t, flags[6].Long)
	require.Empty(t, flags[6].Description)
	require.Empty(t, flags[6].EnvironmentVariable)
	require.Empty(t, flags[6].Default)
}

// Test that the flag definitions are converted from the struct tags into
// objects properly.
func TestConvertDatabaseCLIFlagsToDefinitions(t *testing.T) {
	// Arrange
	pointer := (*DatabaseCLIFlags)(nil)

	// Act
	definitions := pointer.ConvertToCLIFlagDefinitions()

	// Assert
	require.Len(t, definitions, 13)

	definitionMap := make(map[string]*CLIFlagDefinition, len(definitions))
	for _, definition := range definitions {
		definitionMap[definition.Long] = definition
	}

	require.EqualValues(t, "STORK_DATABASE_HOST", definitionMap["db-host"].EnvironmentVariable)
	require.EqualValues(t, "p", definitionMap["db-port"].Short)
	require.EqualValues(t, reflect.Int, definitionMap["db-port"].Kind)
}

// Test that the maintenance flag definitions are converted from the struct tags
// into objects properly.
func TestConvertMaintenanceDatabaseCLIFlagsToDefinitions(t *testing.T) {
	// Arrange
	pointer := (*DatabaseCLIFlagsWithMaintenance)(nil)

	// Act
	definitions := pointer.ConvertToCLIFlagDefinitions()

	// Assert
	require.Len(t, definitions, 13+3)

	definitionMap := make(map[string]*CLIFlagDefinition, len(definitions))
	for _, definition := range definitions {
		definitionMap[definition.Long] = definition
	}

	require.EqualValues(t, "postgres", definitionMap["db-maintenance-user"].Default)
}
