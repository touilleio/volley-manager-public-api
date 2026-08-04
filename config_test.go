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

func TestEnvConfigUsesSnsNotificationDefaults(t *testing.T) {
	previousTopicARN, topicARNWasSet := os.LookupEnv("SNS_TOPIC_ARN")
	previousPublishNewGames, publishNewGamesWasSet := os.LookupEnv("PUBLISH_NEW_GAMES")
	require.NoError(t, os.Unsetenv("SNS_TOPIC_ARN"))
	require.NoError(t, os.Unsetenv("PUBLISH_NEW_GAMES"))
	t.Cleanup(func() {
		if topicARNWasSet {
			require.NoError(t, os.Setenv("SNS_TOPIC_ARN", previousTopicARN))
		} else {
			require.NoError(t, os.Unsetenv("SNS_TOPIC_ARN"))
		}
		if publishNewGamesWasSet {
			require.NoError(t, os.Setenv("PUBLISH_NEW_GAMES", previousPublishNewGames))
		} else {
			require.NoError(t, os.Unsetenv("PUBLISH_NEW_GAMES"))
		}
	})

	var env EnvConfig
	require.NoError(t, envconfig.Process("", &env))
	require.Empty(t, env.SnsTopicARN)
	require.False(t, env.PublishNewGames)
}

func TestEnvConfigParsesSnsNotifications(t *testing.T) {
	t.Setenv("SNS_TOPIC_ARN", "arn:aws:sns:eu-central-1:123:volley")
	t.Setenv("PUBLISH_NEW_GAMES", "true")

	var env EnvConfig
	require.NoError(t, envconfig.Process("", &env))
	require.Equal(t, "arn:aws:sns:eu-central-1:123:volley", env.SnsTopicARN)
	require.True(t, env.PublishNewGames)
}
