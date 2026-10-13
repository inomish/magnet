package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		config         string
		expectedConfig Configuration
		wantErr        bool
	}{
		{
			name:   "should parse config",
			config: `"Alt+Space" = "spawn:kitty"`,
			expectedConfig: Configuration{
				[]ConfigPair{
					{
						Modifiers: ModifierMod1,
						Keysym:    KeysymFromName("Space"),

						Action:   "spawn",
						Argument: "kitty",
					},
				},
			},
			wantErr: false,
		},
		{
			name:    "should not parse when + is missing",
			config:  `"Alt" = "spawn:kitty"`,
			wantErr: true,
		},
		{
			name:    "should not parse when : is missing",
			config:  `"Alt+Space" = "spawnkitty"`,
			wantErr: true,
		},
		{
			name:    "should not parse when first part is not a string",
			config:  `Alt+Space = "spawn:kitty"`,
			wantErr: true,
		},
		{
			name:    "should not parse when second part is not a string",
			config:  `"Alt+Space" = spawn:kitty`,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			testDir := t.TempDir()
			configFilePath := filepath.Join(testDir, "config.toml")

			err := os.WriteFile(configFilePath, []byte(tt.config), 0o600)
			require.NoError(t, err, "could not write test file")

			got, err := ReadConfig(configFilePath)
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)

			assert.Equal(t, tt.expectedConfig, *got)
		})
	}
}
