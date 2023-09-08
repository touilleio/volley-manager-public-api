package main

import (
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
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
		fmt.Println("error:", err)
	}

	for _, game := range games {
		fmt.Printf("Game ID: %d\n", game.GameId)
		if len(game.Referees.Data) == 0 {
			fmt.Println("No referees")
		} else {
			for key, referee := range game.Referees.Data {
				fmt.Printf("Referee %s: %+v\n", key, referee)
			}
		}
		fmt.Println()
	}
}
