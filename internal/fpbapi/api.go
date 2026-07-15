package fpbapi

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/mefrraz/bounce/internal/cache"
	"github.com/mefrraz/bounce/internal/httpclient"
	"github.com/mefrraz/bounce/internal/models"
)

const baseURL = "https://sav2.fpb.pt/api"

type FPBAPI struct {
	http  *httpclient.Client
	cache *cache.Store
}

func New(c *httpclient.Client, s *cache.Store) *FPBAPI {
	return &FPBAPI{http: c, cache: s}
}

func (f *FPBAPI) GetGames(compID, date string) ([]models.Game, error) {
	key := cache.CacheKey("games", compID, date)
	if raw, ok := f.cache.Get(key); ok {
		var games []models.Game
		if err := json.Unmarshal(raw, &games); err == nil {
			return games, nil
		}
	}
	body, err := f.http.Get(fmt.Sprintf("%s/competicoes/%s/jogos?data=%s", baseURL, compID, date))
	if err != nil {
		return nil, err
	}
	var raw []models.Game
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}
	raw2, _ := json.Marshal(raw)
	f.cache.Set(key, raw2, cache.TTLRecent)
	return raw, nil
}

func (f *FPBAPI) GetStandings(compID string) ([]models.Standing, error) {
	key := cache.CacheKey("standings", compID)
	if raw, ok := f.cache.Get(key); ok {
		var s []models.Standing
		if err := json.Unmarshal(raw, &s); err == nil {
			return s, nil
		}
	}
	body, err := f.http.Get(fmt.Sprintf("%s/competicoes/%s/classificacao", baseURL, compID))
	if err != nil {
		return nil, err
	}
	var raw []models.Standing
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}
	raw2, _ := json.Marshal(raw)
	f.cache.Set(key, raw2, cache.TTLStandings)
	return raw, nil
}

func (f *FPBAPI) GetGame(internalID string) (*models.Game, error) {
	key := cache.CacheKey("game", internalID)
	if raw, ok := f.cache.Get(key); ok {
		var g models.Game
		if err := json.Unmarshal(raw, &g); err == nil {
			return &g, nil
		}
	}
	body, err := f.http.Get(fmt.Sprintf("%s/jogos/%s", baseURL, url.PathEscape(internalID)))
	if err != nil {
		return nil, err
	}
	var g models.Game
	if err := json.Unmarshal(body, &g); err != nil {
		return nil, err
	}
	raw2, _ := json.Marshal(g)
	f.cache.Set(key, raw2, cache.TTLRecent)
	return &g, nil
}

func (f *FPBAPI) GetCompetitions() ([]models.Competition, error) {
	key := cache.CacheKey("competitions")
	if raw, ok := f.cache.Get(key); ok {
		var c []models.Competition
		if err := json.Unmarshal(raw, &c); err == nil {
			return c, nil
		}
	}
	body, err := f.http.Get(baseURL + "/competicoes")
	if err != nil {
		return nil, err
	}
	var raw []models.Competition
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}
	raw2, _ := json.Marshal(raw)
	f.cache.Set(key, raw2, 1440)
	return raw, nil
}
