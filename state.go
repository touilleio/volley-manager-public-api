package main

import (
	"encoding/json"
	"sync"
)

type state struct {
	teamsId        []int
	rawGames       []Game
	rawRankings    []GroupRankings
	teams          map[int]Team
	gamesPerTeam   map[int][]Game
	rankingPerTeam map[int]GroupRankings
	leaguePerTeam  map[int]League
	groupPerTeam   map[int]Group
	lock           *sync.RWMutex
}

func newState(teamsId []int) *state {
	internalState := state{
		teamsId: teamsId,
		lock:    &sync.RWMutex{},
	}
	return &internalState
}

func (s *state) isManagedTeam(teamId int) bool {
	if len(s.teamsId) == 0 {
		return true
	}
	for _, id := range s.teamsId {
		if id == teamId {
			return true
		}
	}
	return false
}

type Game struct {
	GameId   int    `json:"gameId"`
	PlayDate string `json:"playDate"`
	Gender   string `json:"gender"`
	Status   int    `json:"status"`
	Teams    struct {
		Home Team `json:"home"`
		Away Team `json:"away"`
	} `json:"teams"`
	League        League     `json:"league"`
	Phase         Phase      `json:"phase"`
	Group         Group      `json:"group"`
	Hall          Hall       `json:"hall"`
	Referees      Referees   `json:"referees"`
	SetResults    SetResults `json:"setResults"`
	ResultSummary []struct {
		WonSetsHomeTeam int    `json:"wonSetsHomeTeam"`
		WonSetsAwayTeam int    `json:"wonSetsAwayTeam"`
		Winner          string `json:"winner"`
	} `json:"resultSummary"`
}

type Team struct {
	TeamId      int    `json:"teamId"`
	Caption     string `json:"caption"`
	ClubId      string `json:"clubId"`
	ClubCaption string `json:"clubCaption"`
}

type League struct {
	LeagueId         int    `json:"leagueId"`
	LeagueCategoryId int    `json:"leagueCategoryId"`
	Caption          string `json:"caption"`
	Translations     struct {
		D      string `json:"d"`
		ShortD string `json:"shortD"`
		F      string `json:"f"`
		ShortF string `json:"shortF"`
	} `json:"translations"`
}

type Phase struct {
	PhaseId      int    `json:"phaseId"`
	Caption      string `json:"caption"`
	Translations struct {
		D      string `json:"d"`
		ShortD string `json:"shortD"`
		F      string `json:"f"`
		ShortF string `json:"shortF"`
	} `json:"translations"`
}

type Group struct {
	GroupId      int    `json:"groupId"`
	Caption      string `json:"caption"`
	Translations struct {
		D      string `json:"d"`
		ShortD string `json:"shortD"`
		F      string `json:"f"`
		ShortF string `json:"shortF"`
	} `json:"translations"`
}

type Hall struct {
	HallId    int     `json:"hallId"`
	Caption   string  `json:"caption"`
	Street    string  `json:"street"`
	Number    string  `json:"number"`
	Zip       string  `json:"zip"`
	City      string  `json:"city"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	PlusCode  string  `json:"plusCode"`
}

type SetResults struct {
	Data map[string]SetResult
}

type SetResult struct {
	Home string `json:"home"`
	Away string `json:"away"`
}

func (r *SetResults) UnmarshalJSON(data []byte) error {
	var setResultsMap map[string]SetResult
	if err := json.Unmarshal(data, &setResultsMap); err == nil {
		r.Data = setResultsMap
	}
	return nil
}

type Referees struct {
	Data map[string]Referee
}

func (r *Referees) UnmarshalJSON(data []byte) error {
	var refereesMap map[string]Referee
	if err := json.Unmarshal(data, &refereesMap); err == nil {
		r.Data = refereesMap
	}
	return nil
}

type Referee struct {
	RefereeId int    `json:"refereeId"`
	LastName  string `json:"lastName"`
	FirstName string `json:"firstName"`
}

type GroupRankings struct {
	LeagueId int           `json:"leagueId"`
	PhaseId  int           `json:"phaseId"`
	GroupId  int           `json:"groupId"`
	Ranking  []TeamRanking `json:"ranking"`
}

type TeamRanking struct {
	Rank          int    `json:"rank"`
	TeamId        int    `json:"teamId"`
	TeamCaption   string `json:"teamCaption"`
	Games         int    `json:"rawGames"`
	Points        int    `json:"points"`
	Wins          int    `json:"wins"`
	WinsClear     int    `json:"winsClear"`
	WinsNarrow    int    `json:"winsNarrow"`
	Defeats       int    `json:"defeats"`
	DefeatsClear  int    `json:"defeatsClear"`
	DefeatsNarrow int    `json:"defeatsNarrow"`
	SetsWon       int    `json:"setsWon"`
	SetsLost      int    `json:"setsLost"`
	BallsWon      int    `json:"ballsWon"`
	BallsLost     int    `json:"ballsLost"`
	IsTeam        bool   `json:"isTeam"`
}
