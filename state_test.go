package main

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReferees_UnmarshalJSON(t *testing.T) {
	jsonData := `
	[
	  {
	    "gameId": 1,
	    "referees": []
	  },
	  {
	    "gameId": 2,
	    "referees": {
	      "1": {
	        "refereeId": 1,
	        "lastName": "a",
	        "firstName": "b"
	      },
	      "2": {
	        "refereeId": 2,
	        "lastName": "c",
	        "firstName": "d"
	      }
	    }
	  }
	]`

	var games []Game
	err := json.Unmarshal([]byte(jsonData), &games)
	assert.Nil(t, err)
	if err != nil {
		t.Log("error:", err)
	}

	for _, game := range games {
		t.Logf("Game ID: %d", game.GameId)
		if len(game.Referees.Data) == 0 {
			t.Log("No referees")
		} else {
			for key, referee := range game.Referees.Data {
				t.Logf("Referee %s: %+v", key, referee)
			}
		}
	}
}

func TestStateIsManagedTeam(t *testing.T) {
	tests := []struct {
		name            string
		clubID          string
		excludedTeamIDs []int
		team            Team
		want            bool
	}{
		{
			name:   "matching club",
			clubID: "906295",
			team:   Team{TeamId: 6631, ClubId: "906295"},
			want:   true,
		},
		{
			name:   "unrelated club",
			clubID: "906295",
			team:   Team{TeamId: 7000, ClubId: "other"},
			want:   false,
		},
		{
			name:            "matching club but excluded team",
			clubID:          "906295",
			excludedTeamIDs: []int{6631},
			team:            Team{TeamId: 6631, ClubId: "906295"},
			want:            false,
		},
		{
			name:            "empty club includes non-excluded team",
			excludedTeamIDs: []int{6632},
			team:            Team{TeamId: 6631, ClubId: "other"},
			want:            true,
		},
		{
			name:            "empty club still excludes team",
			excludedTeamIDs: []int{6631},
			team:            Team{TeamId: 6631, ClubId: "other"},
			want:            false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newState(tt.clubID, tt.excludedTeamIDs, nil)
			assert.Equal(t, tt.want, s.isManagedTeam(tt.team))
		})
	}
}

func TestStateIsCupWhenLeagueCategoryIsConfigured(t *testing.T) {
	s := newState("", nil, []int{4, 8})

	assert.True(t, s.isCup(Game{League: League{LeagueCategoryId: 4, Caption: "League match"}}))
	assert.True(t, s.isCup(Game{League: League{LeagueCategoryId: 8, Caption: "Regional competition"}}))
	assert.False(t, s.isCup(Game{League: League{LeagueCategoryId: 7, Caption: "Cup final"}}))
}

func TestStateRebuildKeepsCupGamesOutOfCompetitionMetadata(t *testing.T) {
	leagueTeam := Team{TeamId: 1, Caption: "League team", ClubId: "club"}
	cupOnlyTeam := Team{TeamId: 2, Caption: "Cup-only team", ClubId: "club"}
	opponent := Team{TeamId: 3, Caption: "Opponent", ClubId: "other"}

	cupGame := Game{
		GameId: 1,
		Teams: struct {
			Home Team `json:"home"`
			Away Team `json:"away"`
		}{Home: leagueTeam, Away: opponent},
		League: League{LeagueId: 100, LeagueCategoryId: 4, Caption: "Cup"},
		Group:  Group{GroupId: 100, Caption: "Cup group"},
	}
	leagueGame := Game{
		GameId: 2,
		Teams: struct {
			Home Team `json:"home"`
			Away Team `json:"away"`
		}{Home: leagueTeam, Away: opponent},
		League: League{LeagueId: 200, LeagueCategoryId: 7, Caption: "League"},
		Group:  Group{GroupId: 200, Caption: "League group"},
	}
	cupOnlyGame := Game{
		GameId: 3,
		Teams: struct {
			Home Team `json:"home"`
			Away Team `json:"away"`
		}{Home: cupOnlyTeam, Away: opponent},
		League: League{LeagueId: 100, LeagueCategoryId: 4, Caption: "Cup"},
		Group:  Group{GroupId: 100, Caption: "Cup group"},
	}
	s := newState("club", nil, []int{4})
	games := []Game{cupGame, leagueGame, cupOnlyGame}
	rankings := []GroupRankings{
		{GroupId: 100, Ranking: []TeamRanking{{TeamId: leagueTeam.TeamId}}},
		{GroupId: 200, Ranking: []TeamRanking{{TeamId: leagueTeam.TeamId}}},
		{GroupId: 100, Ranking: []TeamRanking{{TeamId: cupOnlyTeam.TeamId}}},
	}

	allGames := s.rebuildManagedGames(games, rankings)

	assert.Len(t, allGames, 3)
	assert.Len(t, s.rawGames, 3)
	assert.Len(t, s.gamesPerTeam[leagueTeam.TeamId], 2)
	assert.Len(t, s.gamesPerTeam[cupOnlyTeam.TeamId], 1)
	assert.Equal(t, leagueGame.League, s.leaguePerTeam[leagueTeam.TeamId])
	assert.Equal(t, leagueGame.Group, s.groupPerTeam[leagueTeam.TeamId])
	assert.NotContains(t, s.leaguePerTeam, cupOnlyTeam.TeamId)
	assert.NotContains(t, s.groupPerTeam, cupOnlyTeam.TeamId)
	assert.Equal(t, leagueGame.Group.GroupId, s.rankingPerTeam[leagueTeam.TeamId].GroupId)
	assert.NotContains(t, s.rankingPerTeam, cupOnlyTeam.TeamId)
}
