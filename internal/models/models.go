package models

type Game struct {
	ID            string `json:"id"`
	Date          string `json:"data"`
	Time          string `json:"hora"`
	HomeTeam      string `json:"equipa_casa"`
	HomeTeamID    string `json:"equipa_casa_id,omitempty"`
	AwayTeam      string `json:"equipa_fora"`
	AwayTeamID    string `json:"equipa_fora_id,omitempty"`
	HomeScore     *int   `json:"resultado_casa"`
	AwayScore     *int   `json:"resultado_fora"`
	Venue         string `json:"local,omitempty"`
	Competition   string `json:"competicao,omitempty"`
	CompetitionID string `json:"competicao_id,omitempty"`
	Journey       string `json:"jornada,omitempty"`
	Status        string `json:"estado"`
	HomeLogo      string `json:"logo_casa,omitempty"`
	AwayLogo      string `json:"logo_fora,omitempty"`
	HomeClubID    int    `json:"clube_casa_id,omitempty"`
	AwayClubID    int    `json:"clube_fora_id,omitempty"`
	Category      string `json:"escalao,omitempty"`
	Season        string `json:"epoca,omitempty"`
}

type Standing struct {
	Position      int    `json:"posicao"`
	Team          string `json:"equipa"`
	TeamID        string `json:"equipa_id,omitempty"`
	ClubID        int    `json:"clube_id,omitempty"`
	Played        int    `json:"j"`
	Won           int    `json:"v"`
	Lost          int    `json:"d"`
	PointsFor     *int   `json:"pm"`
	PointsAgainst *int   `json:"ps"`
	Diff          *int   `json:"dif"`
	Points        int    `json:"pts"`
	Logo          string `json:"logo,omitempty"`
}

type Competition struct {
	ID           string `json:"id"`
	Name         string `json:"nome"`
	Abbreviation string `json:"abreviatura,omitempty"`
	Logo         string `json:"logo,omitempty"`
	Association  string `json:"associacao,omitempty"`
	Category     string `json:"escalao,omitempty"`
	Season       string `json:"epoca,omitempty"`
}

type GameDetail struct {
	Game
}
