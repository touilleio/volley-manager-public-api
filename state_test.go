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
			s := newState(tt.clubID, tt.excludedTeamIDs)
			assert.Equal(t, tt.want, s.isManagedTeam(tt.team))
		})
	}
}
