package main

import (
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
