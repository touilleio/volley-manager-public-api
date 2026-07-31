package main

import "log/slog"

type managedGames struct {
	allGames       []Game
	teams          map[int]Team
	gamesPerTeam   map[int][]Game
	rankingPerTeam map[int]GroupRankings
	leaguePerTeam  map[int]League
	groupPerTeam   map[int]Group
}

func newManagedGames(gameCount int) managedGames {
	return managedGames{
		allGames:       make([]Game, 0, gameCount),
		teams:          make(map[int]Team),
		gamesPerTeam:   make(map[int][]Game),
		rankingPerTeam: make(map[int]GroupRankings),
		leaguePerTeam:  make(map[int]League),
		groupPerTeam:   make(map[int]Group),
	}
}

// rebuildManagedGames filters freshly fetched collections and publishes them
// as one consistent snapshot under a single lock acquisition. Readers never
// observe unfiltered upstream data.
func (s *state) rebuildManagedGames(games []Game, rankings []GroupRankings) []Game {
	managed := newManagedGames(len(games))
	for _, game := range games {
		isManaged := s.addManagedTeam(&managed, game, game.Teams.Away)
		isManaged = s.addManagedTeam(&managed, game, game.Teams.Home) || isManaged
		if isManaged {
			managed.allGames = append(managed.allGames, game)
		}
	}

	for _, ranking := range rankings {
		for teamID, group := range managed.groupPerTeam {
			if group.GroupId != ranking.GroupId {
				continue
			}
			managed.rankingPerTeam[teamID] = ranking
			for index, teamRanking := range ranking.Ranking {
				if teamRanking.TeamId == teamID {
					teamRanking.IsTeam = true
					ranking.Ranking[index] = teamRanking
				}
			}
			slog.Debug("Team is in this ranking", "team", managed.teams[teamID].Caption, "leagueId", ranking.LeagueId, "phaseId", ranking.PhaseId, "groupId", ranking.GroupId, "rankingSize", len(ranking.Ranking))
			slog.Debug("Ranking detail", "ranking", ranking)
		}
	}

	s.lock.Lock()
	s.rawGames = managed.allGames
	s.teams = managed.teams
	s.gamesPerTeam = managed.gamesPerTeam
	s.rankingPerTeam = managed.rankingPerTeam
	s.leaguePerTeam = managed.leaguePerTeam
	s.groupPerTeam = managed.groupPerTeam
	s.lock.Unlock()

	return managed.allGames
}

func (s *state) addManagedTeam(managed *managedGames, game Game, team Team) bool {
	if !s.isManagedTeam(team) {
		return false
	}

	managed.teams[team.TeamId] = team
	managed.gamesPerTeam[team.TeamId] = append(managed.gamesPerTeam[team.TeamId], game)
	if s.isCup(game) {
		return true
	}

	if league, exists := managed.leaguePerTeam[team.TeamId]; exists {
		if league.LeagueId != game.League.LeagueId {
			slog.Warn("League mismatch for team", "team", team.Caption, "previous", league.Caption, "current", game.League.Caption)
		}
	} else {
		managed.leaguePerTeam[team.TeamId] = game.League
	}
	if group, exists := managed.groupPerTeam[team.TeamId]; exists {
		if group.GroupId != game.Group.GroupId {
			slog.Warn("Group mismatch for team", "team", team.Caption, "previous", group.Caption, "current", game.Group.Caption)
		}
	} else {
		managed.groupPerTeam[team.TeamId] = game.Group
	}

	return true
}
