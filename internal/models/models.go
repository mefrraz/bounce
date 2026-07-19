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
	HomeAbbrev  string       `json:"abrev_casa,omitempty"`
	AwayAbbrev  string       `json:"abrev_fora,omitempty"`
	Phase       string       `json:"fase,omitempty"`
	Attendance  int          `json:"espetadores"`
	PavilionID  string       `json:"recinto_id,omitempty"`
	Periods     []Period     `json:"periodos,omitempty"`
	HomeStats   []PlayerStat `json:"stats_casa,omitempty"`
	AwayStats   []PlayerStat `json:"stats_fora,omitempty"`
	GameLeaders  []GameLeader  `json:"game_leaders,omitempty"`
	TopPerfCasa  TopPerformer  `json:"top_perf_casa"`
	TopPerfFora  TopPerformer  `json:"top_perf_fora"`
	TopPerfStats []TopPerfStat `json:"top_perf_stats"`
}

type Period struct {
	Number    int `json:"periodo"`
	HomeScore int `json:"casa"`
	AwayScore int `json:"fora"`
}

type GameLeader struct {
	Category string         `json:"categoria"`
	Home     LeaderPlayer   `json:"casa"`
	Away     LeaderPlayer   `json:"fora"`
}

type LeaderPlayer struct {
	Name  string `json:"nome"`
	Stat  string `json:"valor"`
	Photo string `json:"foto,omitempty"`
}

type TopPerformer struct {
	Name  string `json:"nome"`
	Photo string `json:"foto,omitempty"`
}

type TopPerfStat struct {
	Label string `json:"label"`
	Casa  string `json:"casa"`
	Fora  string `json:"fora"`
}

type PlayerStat struct {
	Name     string `json:"nome"`
	Photo    string `json:"foto,omitempty"`
	Number   int    `json:"numero,omitempty"`
	MIN      string `json:"min,omitempty"`
	PTS      int    `json:"pts"`
	L2       string `json:"l2,omitempty"`
	L2Pct    string `json:"l2pct,omitempty"`
	L3       string `json:"l3,omitempty"`
	L3Pct    string `json:"l3pct,omitempty"`
	LL       string `json:"ll,omitempty"`
	LLPct    string `json:"llpct,omitempty"`
	RO       int    `json:"ro,omitempty"`
	RD       int    `json:"rd,omitempty"`
	RT       int    `json:"rt,omitempty"`
	AS       int    `json:"as,omitempty"`
	RB       int    `json:"rb,omitempty"`
	TO       int    `json:"to,omitempty"`
	DL       int    `json:"dl,omitempty"`
	FC       int    `json:"fc,omitempty"`
	FS       int    `json:"fs,omitempty"`
	PlusMinus int   `json:"mais_menos,omitempty"`
	VAL      int    `json:"val,omitempty"`
}

type Team struct {
	ID           string `json:"equipa_id"`
	ClubID       int    `json:"clube_id,omitempty"`
	Name         string `json:"nome"`
	Abbreviation string `json:"abreviatura,omitempty"`
	Logo         string `json:"logo,omitempty"`
	Photo        string `json:"photo,omitempty"`
	Association  string `json:"associacao,omitempty"`
}
