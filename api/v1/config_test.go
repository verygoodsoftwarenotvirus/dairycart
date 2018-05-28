package api

import (
	"database/sql"
	"os"
	"testing"

	"github.com/dairycart/dairycart/storage/v1/database/mock"
	"github.com/dairycart/dairycart/storage/v1/images/mock"

	"github.com/dchest/uniuri"
	"github.com/go-chi/chi"
	"github.com/lib/pq"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const (
	exampleConnStr                = "postgres://dairytest:hunter2@database:5432/dairytest?sslmode=disable"
	exampleDatabasePluginPath     = "example_files/plugins/mock_db.so"
	exampleImageStoragePluginPath = "example_files/plugins/mock_img.so"
	exampleInvalidTomlFile        = "example_files/configs/bad_config.toml"
)

func init() {
	sql.Register("nothing", &pq.Driver{})
	os.Setenv("TESTING", "true")
}

func TestLoadPlugin(t *testing.T) {
	t.Parallel()

	unempty := "not empty lol"

	t.Run("normal operation", func(_t *testing.T) {
		_t.Parallel()

		actual, err := loadPlugin(exampleDatabasePluginPath, "Example")

		assert.NotNil(_t, actual)
		assert.NoError(_t, err)
	})

	t.Run("with lowercased symbol name", func(_t *testing.T) {
		_t.Parallel()

		actual, err := loadPlugin(exampleDatabasePluginPath, "example")

		assert.NotNil(_t, actual)
		assert.NoError(_t, err)
	})

	t.Run("with empty plugin path", func(_t *testing.T) {
		_t.Parallel()

		actual, err := loadPlugin("", unempty)
		assert.Nil(_t, actual)
		assert.Error(_t, err)
	})

	t.Run("with empty symbol name", func(_t *testing.T) {
		_t.Parallel()

		actual, err := loadPlugin(unempty, "")
		assert.Nil(_t, actual)
		assert.Error(_t, err)
	})

	t.Run("with error opening plugin", func(_t *testing.T) {
		_t.Parallel()

		actual, err := loadPlugin("invalid path", unempty)
		assert.Nil(_t, actual)
		assert.Error(_t, err)
	})

	t.Run("with error looking up symbol", func(_t *testing.T) {
		_t.Parallel()

		actual, err := loadPlugin(exampleDatabasePluginPath, "nonexistent")

		assert.Nil(_t, actual)
		assert.Error(_t, err)
	})
}

func TestSetupCookieStorage(t *testing.T) {
	t.Parallel()

	t.Run("normal operation", func(_t *testing.T) {
		_t.Parallel()
		cs, err := setupCookieStorage("arbitrarily long secret for testing purposes")
		assert.NoError(_t, err)
		assert.NotNil(_t, cs)
	})

	t.Run("with short secret", func(_t *testing.T) {
		_t.Parallel()
		cs, err := setupCookieStorage("lol")
		assert.Error(_t, err)
		assert.Nil(_t, cs)
	})
}

func TestSetConfigDefaults(t *testing.T) {
	t.Parallel()

	t.Run("normal operation", func(_t *testing.T) {
		_t.Parallel()

		actual := viper.New()
		setConfigDefaults(actual, "")

		assert.Equal(_t, actual.GetInt(portKey), defaultPort, "default port should be set")
		assert.False(_t, actual.GetBool(migrateExampleDataKey), "default migrate example data value should be set")
		assert.Equal(_t, actual.GetString(domainKey),
			"http://localhost:4321", "default domain should be set")
		assert.Equal(_t, actual.GetString(databaseTypeKey), DefaultDatabaseProvider, "default database provider should be set")
		assert.Equal(_t, actual.GetString(imageStorageTypeKey), DefaultImageStorageProvider, "default _ should be set")
		assert.NotEmpty(_t, actual.GetString(secretKey), "default secret should be autogenerated.")
	})
}

func TestLoadServerConfig(t *testing.T) {
	t.Run("normal operation", func(_t *testing.T) {
		actual, err := LoadServerConfig("")
		require.NotNil(_t, actual)
		assert.NoError(_t, err)

		assert.Equal(_t, actual.GetInt(portKey), defaultPort, "default port should be set")
		assert.False(_t, actual.GetBool(migrateExampleDataKey), "default migrate example data value should be set")
		assert.Equal(_t, actual.GetString(domainKey),
			"http://localhost:4321", "default domain should be set")
		assert.Equal(_t, actual.GetString(databaseTypeKey), DefaultDatabaseProvider, "default database provider should be set")
		assert.Equal(_t, actual.GetString(imageStorageTypeKey), DefaultImageStorageProvider, "default _ should be set")
		assert.NotEmpty(_t, actual.GetString(secretKey), "default secret should be autogenerated.")
	})

	t.Run("with error reading config file", func(_t *testing.T) {
		_, err := LoadServerConfig(exampleInvalidTomlFile)
		assert.Error(_t, err)
	})
}

func TestBuildServerConfig(t *testing.T) {
	t.Parallel()

	t.Run("normal operation", func(_t *testing.T) {
		_t.Parallel()

		tmpSecret := uniuri.NewLen(mandatorySecretLength)
		config := viper.New()
		config.Set(databaseConnectionKey, exampleConnStr)
		config.Set(databaseTypeKey, DefaultDatabaseProvider)
		config.Set(imageStorageTypeKey, DefaultImageStorageProvider)
		config.Set(secretKey, tmpSecret)

		actual, err := BuildServerConfig(config)
		assert.NotNil(_t, actual)
		assert.NoError(_t, err)
	})

	t.Run("with error building database configuration", func(_t *testing.T) {
		_t.Parallel()

		tmpSecret := uniuri.NewLen(mandatorySecretLength)
		config := viper.New()
		config.Set(imageStorageTypeKey, DefaultImageStorageProvider)
		config.Set(secretKey, tmpSecret)

		actual, err := BuildServerConfig(config)
		assert.Nil(_t, actual)
		assert.Error(_t, err)
	})

	t.Run("with error loading image storage", func(_t *testing.T) {
		_t.Parallel()

		tmpSecret := uniuri.NewLen(mandatorySecretLength)
		config := viper.New()
		config.Set(databaseConnectionKey, exampleConnStr)
		config.Set(databaseTypeKey, DefaultDatabaseProvider)
		config.Set(secretKey, tmpSecret)

		actual, err := BuildServerConfig(config)
		assert.Nil(_t, actual)
		assert.Error(_t, err)
	})

	t.Run("with error setting up cookie storage", func(_t *testing.T) {
		_t.Parallel()

		config := viper.New()
		config.Set(databaseConnectionKey, exampleConnStr)
		config.Set(databaseTypeKey, DefaultDatabaseProvider)
		config.Set(imageStorageTypeKey, DefaultImageStorageProvider)

		actual, err := BuildServerConfig(config)
		assert.Nil(_t, actual)
		assert.Error(_t, err)
	})
}

func TestBuildDatabaseFromConfig(t *testing.T) {
	t.Parallel()

	t.Run("normal operation", func(_t *testing.T) {
		_t.Parallel()

		cfg := viper.New()
		cfg.Set(databaseConnectionKey, exampleConnStr)
		cfg.Set(databaseTypeKey, DefaultDatabaseProvider)

		_, _, err := buildDatabaseFromConfig(cfg)

		assert.NoError(_t, err)
	})

	t.Run("with missing database key", func(_t *testing.T) {
		_t.Parallel()

		cfg := viper.New()
		cfg.Set(databaseConnectionKey, exampleConnStr)
		_, _, err := buildDatabaseFromConfig(cfg)

		assert.Error(_t, err)
	})

	t.Run("with empty connection key", func(_t *testing.T) {
		_t.Parallel()

		cfg := viper.New()
		cfg.Set(databaseConnectionKey, "")
		cfg.Set(databaseTypeKey, DefaultDatabaseProvider)
		_, _, err := buildDatabaseFromConfig(cfg)

		assert.Error(_t, err)
	})

	t.Run("with missing plugin key", func(_t *testing.T) {
		_t.Parallel()

		cfg := viper.New()
		cfg.Set(databaseConnectionKey, exampleConnStr)
		cfg.Set(databaseTypeKey, "nothing")

		_, _, err := buildDatabaseFromConfig(cfg)

		assert.Error(_t, err)
	})

	t.Run("with empty plugin key path", func(_t *testing.T) {
		_t.Parallel()

		cfg := viper.New()
		cfg.Set(databaseConnectionKey, exampleConnStr)
		cfg.Set(databaseTypeKey, "nothing")
		cfg.Set(databasePluginKey, "")

		_, _, err := buildDatabaseFromConfig(cfg)

		assert.Error(_t, err)
	})

	t.Run("with error loading plugin", func(_t *testing.T) {
		_t.Parallel()

		cfg := viper.New()
		cfg.Set(databaseConnectionKey, exampleConnStr)
		cfg.Set(databaseTypeKey, "nothing")
		cfg.Set(databasePluginKey, exampleImageStoragePluginPath)

		_, _, err := buildDatabaseFromConfig(cfg)

		assert.Error(_t, err)
	})
}

func TestLoadDatabasePlugin(t *testing.T) {
	t.Parallel()

	t.Run("normal operation", func(_t *testing.T) {
		_t.Parallel()

		actual, err := loadDatabasePlugin(exampleDatabasePluginPath, "Example")

		assert.NotNil(_t, actual)
		assert.NoError(_t, err)
	})

	t.Run("with error loading plugin", func(_t *testing.T) {
		_t.Parallel()

		actual, err := loadDatabasePlugin("", "")

		assert.Nil(_t, actual)
		assert.Error(_t, err)
	})

	t.Run("with error loading symbol", func(_t *testing.T) {
		_t.Parallel()

		actual, err := loadDatabasePlugin(exampleImageStoragePluginPath, "Example")

		assert.Nil(_t, actual)
		assert.Error(_t, err)
	})
}

func TestBuildImageStorerFromConfig(t *testing.T) {
	t.Parallel()

	t.Run("normal operation", func(_t *testing.T) {
		_t.Parallel()

		cfg := viper.New()
		cfg.Set(imageStorageTypeKey, DefaultImageStorageProvider)

		_, err := buildImageStorerFromConfig(cfg)
		assert.NoError(_t, err)
	})

	t.Run("with missing storage key", func(_t *testing.T) {
		_t.Parallel()

		_, err := buildImageStorerFromConfig(viper.New())
		assert.Error(_t, err)
	})

	t.Run("with missing plugin key", func(_t *testing.T) {
		_t.Parallel()

		cfg := viper.New()
		cfg.Set(imageStorageTypeKey, "nothing")

		_, err := buildImageStorerFromConfig(cfg)
		assert.Error(_t, err)
	})

	t.Run("with empty plugin key path", func(_t *testing.T) {
		_t.Parallel()

		cfg := viper.New()
		cfg.Set(imageStorageTypeKey, "nothing")
		cfg.Set(imageStoragePluginKey, "")

		_, err := buildImageStorerFromConfig(cfg)
		assert.Error(_t, err)
	})

	t.Run("with error loading plugin", func(_t *testing.T) {
		_t.Parallel()

		cfg := viper.New()
		cfg.Set(imageStorageTypeKey, "nothing")
		cfg.Set(imageStoragePluginKey, exampleDatabasePluginPath)

		_, err := buildImageStorerFromConfig(cfg)
		assert.Error(_t, err)
	})
}

func TestLoadImageStoragePlugin(t *testing.T) {
	t.Parallel()

	t.Run("normal operation", func(_t *testing.T) {
		_t.Parallel()

		actual, err := loadImageStoragePlugin(exampleImageStoragePluginPath, "Example")

		assert.NotNil(_t, actual)
		assert.NoError(_t, err)
	})

	t.Run("with error loading plugin", func(_t *testing.T) {
		_t.Parallel()

		actual, err := loadImageStoragePlugin("", "")

		assert.Nil(_t, actual)
		assert.Error(_t, err)
	})

	t.Run("with error loading symbol", func(_t *testing.T) {
		_t.Parallel()

		actual, err := loadImageStoragePlugin(exampleDatabasePluginPath, "Example")

		assert.Nil(_t, actual)
		assert.Error(_t, err)
	})
}

func TestInitializeServerComponents(t *testing.T) {
	t.Parallel()

	t.Run("normal use case", func(_t *testing.T) {
		_t.Parallel()

		mis := &imgmock.MockImageStorer{}
		mis.On("Init", mock.Anything, mock.Anything).Return(nil)

		mdb := &dairymock.MockDB{}
		mdb.On("Migrate", mock.Anything, mock.Anything, mock.Anything).Return(nil)

		config := &ServerConfig{
			ImageStorer:    mis,
			DatabaseClient: mdb,
			Router:         chi.NewMux(),
		}

		cfg := viper.New()
		cfg.Set(databaseConnectionKey, "blah blah blah")
		cfg.Set(migrateExampleDataKey, true)

		err := InitializeServerComponents(cfg, config)
		assert.NoError(_t, err)
	})

	t.Run("with error initializing image storage", func(_t *testing.T) {
		_t.Parallel()

		mis := &imgmock.MockImageStorer{}
		mis.On("Init", mock.Anything, mock.Anything).Return(generateArbitraryError())

		mdb := &dairymock.MockDB{}
		mdb.On("Migrate", mock.Anything, mock.Anything, mock.Anything).Return(nil)

		config := &ServerConfig{
			ImageStorer:    mis,
			DatabaseClient: mdb,
			Router:         chi.NewMux(),
		}

		cfg := viper.New()
		cfg.Set(databaseConnectionKey, "blah blah blah")
		cfg.Set(migrateExampleDataKey, true)

		err := InitializeServerComponents(cfg, config)
		assert.Error(_t, err)
	})

	t.Run("with error migrating database", func(_t *testing.T) {
		_t.Parallel()

		mis := &imgmock.MockImageStorer{}
		mis.On("Init", mock.Anything, mock.Anything).Return(nil)

		mdb := &dairymock.MockDB{}
		mdb.On("Migrate", mock.Anything, mock.Anything, mock.Anything).Return(generateArbitraryError())

		config := &ServerConfig{
			ImageStorer:    mis,
			DatabaseClient: mdb,
			Router:         chi.NewMux(),
		}

		cfg := viper.New()
		cfg.Set(databaseConnectionKey, "blah blah blah")
		cfg.Set(migrateExampleDataKey, true)

		err := InitializeServerComponents(cfg, config)
		assert.Error(_t, err)
	})
}