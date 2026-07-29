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
