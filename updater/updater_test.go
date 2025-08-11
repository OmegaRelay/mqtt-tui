package updater_test

import (
	"testing"

	"github.com/OmegaRelay/mqtt-tui/updater"
	"github.com/stretchr/testify/require"
)

func TestIsUpdateAvailable(t *testing.T) {
	updater := updater.New("OmegaRelay/mqtt-tui", updater.Version{})

	updateAvailable, err := updater.IsUpdateAvailable()
	require.NoError(t, err)
	require.Equal(t, true, updateAvailable)
}

func TestGetChangelog(t *testing.T) {
}

func TestInstallLatestRelease(t *testing.T) {
	updater := updater.New("OmegaRelay/mqtt-tui", updater.Version{})
	err := updater.InstallLatestUpdate()
	require.NoError(t, err)
}
