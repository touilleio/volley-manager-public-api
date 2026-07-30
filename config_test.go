package main

import (
	"os"
	"testing"

	"github.com/kelseyhightower/envconfig"
	"github.com/stretchr/testify/require"
)

func TestEnvConfigParsesClubSelection(t *testing.T) {
	t.Setenv("CLUB_ID", "906295")
	t.Setenv("EXCLUDED_TEAMS_ID", "6631,6632")

	var env EnvConfig
	require.NoError(t, envconfig.Process("", &env))
	require.Equal(t, "906295", env.ClubID)
	require.Equal(t, []int{6631, 6632}, env.ExcludedTeamIDs)
}

func TestEnvConfigAllowsEmptyClubSelection(t *testing.T) {
	t.Setenv("CLUB_ID", "")
	t.Setenv("EXCLUDED_TEAMS_ID", "")

	var env EnvConfig
	require.NoError(t, envconfig.Process("", &env))
	require.Empty(t, env.ClubID)
	require.Empty(t, env.ExcludedTeamIDs)
}

func TestEnvConfigUsesDefaultCupLeagueCategoryIDs(t *testing.T) {
	previousValue, wasSet := os.LookupEnv("CUP_LEAGUE_CATEGORY_IDS")
	require.NoError(t, os.Unsetenv("CUP_LEAGUE_CATEGORY_IDS"))
	t.Cleanup(func() {
		if wasSet {
			require.NoError(t, os.Setenv("CUP_LEAGUE_CATEGORY_IDS", previousValue))
			return
		}
		require.NoError(t, os.Unsetenv("CUP_LEAGUE_CATEGORY_IDS"))
	})

	var env EnvConfig
	require.NoError(t, envconfig.Process("", &env))
	require.Equal(t, []int{4}, env.CupLeagueCategoryIDs)
}

func TestEnvConfigParsesCupLeagueCategoryIDs(t *testing.T) {
	t.Setenv("CUP_LEAGUE_CATEGORY_IDS", "4,8")

	var env EnvConfig
	require.NoError(t, envconfig.Process("", &env))
	require.Equal(t, []int{4, 8}, env.CupLeagueCategoryIDs)
}
