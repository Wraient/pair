package tracker

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/browser"
	"github.com/wraient/pair/pkg/database"
)

const (
	malAPIBaseURL        = "https://api.myanimelist.net/v2"
	malOAuthURL          = "https://myanimelist.net/v1/oauth2"
	malOAuthAuthorizeURL = malOAuthURL + "/authorize"
	malOAuthTokenURL     = malOAuthURL + "/token"
	malClientID          = "" // Set your client ID here
	malClientSecret      = "" // Set your client secret here
	malTokenFilename     = "mal_token.json"
	malStateFilename     = "mal_state.txt"
	malRedirectURI       = "http://localhost:8000/oauth/callback"
	malServerPort        = 8000
)

// MALToken represents the OAuth token response from MyAnimeList
type MALToken struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
	ExpiresAt    time.Time
}

// MALTracker implements the Tracker interface for MyAnimeList
type MALTracker struct {
	token      *MALToken
	tokenPath  string
	statePath  string
	httpClient *http.Client
}

// NewMALTracker creates a new MALTracker
func NewMALTracker(configDir string) *MALTracker {
	// Create config directory if it doesn't exist
	if err := os.MkdirAll(configDir, 0755); err != nil {
		fmt.Printf("Failed to create config directory: %v\n", err)
	}

	return &MALTracker{
		tokenPath:  filepath.Join(configDir, malTokenFilename),
		statePath:  filepath.Join(configDir, malStateFilename),
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// Name returns the name of the tracker
func (t *MALTracker) Name() string {
	return "mal"
}

// IsAuthenticated checks if the user is authenticated with the tracker
func (t *MALTracker) IsAuthenticated() bool {
	if t.token == nil {
		t.loadToken()
	}
	return t.token != nil && t.token.AccessToken != "" && time.Now().Before(t.token.ExpiresAt)
}

// loadToken loads the token from the token file
func (t *MALTracker) loadToken() error {
	data, err := os.ReadFile(t.tokenPath)
	if err != nil {
		return err
	}

	var token MALToken
	if err := json.Unmarshal(data, &token); err != nil {
		return err
	}

	t.token = &token
	return nil
}

// saveToken saves the token to the token file
func (t *MALTracker) saveToken() error {
	data, err := json.Marshal(t.token)
	if err != nil {
		return err
	}

	return os.WriteFile(t.tokenPath, data, 0600)
}

// refreshToken refreshes the access token using the refresh token
func (t *MALTracker) refreshToken() error {
	if t.token == nil || t.token.RefreshToken == "" {
		return fmt.Errorf("no refresh token available")
	}

	data := url.Values{}
	data.Set("client_id", malClientID)
	data.Set("client_secret", malClientSecret)
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", t.token.RefreshToken)

	req, err := http.NewRequest("POST", malOAuthTokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create refresh token request: %w", err)
	}

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	resp, err := t.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to refresh token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to refresh token: %s (%d)", string(body), resp.StatusCode)
	}

	var token MALToken
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return fmt.Errorf("failed to decode token response: %w", err)
	}

	token.ExpiresAt = time.Now().Add(time.Duration(token.ExpiresIn) * time.Second)
	t.token = &token

	return t.saveToken()
}

// Authenticate authenticates the user with MyAnimeList
func (t *MALTracker) Authenticate(ctx context.Context) error {
	// Try to load existing token
	if err := t.loadToken(); err == nil && t.IsAuthenticated() {
		return nil
	}

	// Try to refresh token if available
	if t.token != nil && t.token.RefreshToken != "" {
		if err := t.refreshToken(); err == nil {
			return nil
		}
		// If refresh fails, continue with new authentication
	}

	// Generate a state value to prevent CSRF
	state := fmt.Sprintf("%d", time.Now().UnixNano())
	if err := os.WriteFile(t.statePath, []byte(state), 0600); err != nil {
		return fmt.Errorf("failed to save state: %w", err)
	}

	// Construct the authorization URL
	authURL := fmt.Sprintf("%s?response_type=code&client_id=%s&state=%s&redirect_uri=%s&code_challenge_method=plain&code_challenge=%s",
		malOAuthAuthorizeURL,
		url.QueryEscape(malClientID),
		url.QueryEscape(state),
		url.QueryEscape(malRedirectURI),
		url.QueryEscape(malClientID),
	)

	// Open the browser to the authorization URL
	fmt.Printf("Opening browser to authenticate with MyAnimeList...\n")
	if err := browser.OpenURL(authURL); err != nil {
		return fmt.Errorf("failed to open browser: %w", err)
	}

	// Start a local server to handle the callback
	callbackCh := make(chan string)
	errCh := make(chan error)

	server := &http.Server{
		Addr: fmt.Sprintf(":%d", malServerPort),
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/oauth/callback" {
				http.NotFound(w, r)
				return
			}

			// Get the authorization code and state from the callback
			code := r.URL.Query().Get("code")
			returnedState := r.URL.Query().Get("state")

			// Verify the state to prevent CSRF
			savedState, err := os.ReadFile(t.statePath)
			if err != nil || string(savedState) != returnedState {
				errCh <- fmt.Errorf("invalid state")
				http.Error(w, "Invalid state", http.StatusBadRequest)
				return
			}

			// Send the authorization code to the channel
			callbackCh <- code

			// Display a success message
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(`
				<html>
					<body>
						<h1>Authentication Successful</h1>
						<p>You can now close this window and return to the application.</p>
						<script>window.close();</script>
					</body>
				</html>
			`))
		}),
	}

	// Start the server in a goroutine
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	// Wait for the callback or context cancellation
	var code string
	select {
	case code = <-callbackCh:
		// Continue
	case err := <-errCh:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}

	// Shutdown the server
	go server.Shutdown(context.Background())

	// Exchange the authorization code for an access token
	data := url.Values{}
	data.Set("client_id", malClientID)
	data.Set("client_secret", malClientSecret)
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("redirect_uri", malRedirectURI)
	data.Set("code_verifier", malClientID)

	req, err := http.NewRequest("POST", malOAuthTokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create token request: %w", err)
	}

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	resp, err := t.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to get token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to get token: %s (%d)", string(body), resp.StatusCode)
	}

	var token MALToken
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return fmt.Errorf("failed to decode token response: %w", err)
	}

	token.ExpiresAt = time.Now().Add(time.Duration(token.ExpiresIn) * time.Second)
	t.token = &token

	return t.saveToken()
}

// apiRequest makes an authenticated request to the MAL API
func (t *MALTracker) apiRequest(ctx context.Context, method, path string, query url.Values, body io.Reader) (*http.Response, error) {
	if !t.IsAuthenticated() {
		return nil, fmt.Errorf("not authenticated")
	}

	u, err := url.Parse(malAPIBaseURL + path)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}

	if query != nil {
		u.RawQuery = query.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, method, u.String(), body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Add("Authorization", fmt.Sprintf("%s %s", t.token.TokenType, t.token.AccessToken))

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	if resp.StatusCode == http.StatusUnauthorized {
		// Token expired, try to refresh
		if err := t.refreshToken(); err != nil {
			resp.Body.Close()
			return nil, fmt.Errorf("failed to refresh token: %w", err)
		}

		// Retry the request with the new token
		resp.Body.Close()
		req.Header.Set("Authorization", fmt.Sprintf("%s %s", t.token.TokenType, t.token.AccessToken))
		return t.httpClient.Do(req)
	}

	return resp, nil
}

// SearchAnime searches for anime on MyAnimeList
func (t *MALTracker) SearchAnime(ctx context.Context, query string, limit int) ([]AnimeInfo, error) {
	q := url.Values{}
	q.Set("q", query)
	q.Set("limit", strconv.Itoa(limit))
	q.Set("fields", "id,title,alternative_titles,main_picture,synopsis,mean,status,genres,media_type,num_episodes,start_season,studios")

	resp, err := t.apiRequest(ctx, "GET", "/anime", q, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to search anime: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to search anime: %s (%d)", string(body), resp.StatusCode)
	}

	var result struct {
		Data []struct {
			Node struct {
				ID                int    `json:"id"`
				Title             string `json:"title"`
				AlternativeTitles struct {
					English  string   `json:"en"`
					Japanese string   `json:"ja"`
					Synonyms []string `json:"synonyms"`
				} `json:"alternative_titles"`
				MainPicture struct {
					Medium string `json:"medium"`
					Large  string `json:"large"`
				} `json:"main_picture"`
				Synopsis string  `json:"synopsis"`
				Mean     float64 `json:"mean"`
				Status   string  `json:"status"`
				Genres   []struct {
					ID   int    `json:"id"`
					Name string `json:"name"`
				} `json:"genres"`
				MediaType   string `json:"media_type"`
				NumEpisodes int    `json:"num_episodes"`
				StartSeason struct {
					Year   int    `json:"year"`
					Season string `json:"season"`
				} `json:"start_season"`
				Studios []struct {
					ID   int    `json:"id"`
					Name string `json:"name"`
				} `json:"studios"`
			} `json:"node"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode search results: %w", err)
	}

	animes := make([]AnimeInfo, 0, len(result.Data))
	for _, item := range result.Data {
		node := item.Node
		genres := make([]string, 0, len(node.Genres))
		for _, g := range node.Genres {
			genres = append(genres, g.Name)
		}

		studios := make([]string, 0, len(node.Studios))
		for _, s := range node.Studios {
			studios = append(studios, s.Name)
		}

		var alternativeTitles []string
		if node.AlternativeTitles.English != "" {
			alternativeTitles = append(alternativeTitles, node.AlternativeTitles.English)
		}
		if node.AlternativeTitles.Japanese != "" {
			alternativeTitles = append(alternativeTitles, node.AlternativeTitles.Japanese)
		}
		alternativeTitles = append(alternativeTitles, node.AlternativeTitles.Synonyms...)

		anime := AnimeInfo{
			ID:                strconv.Itoa(node.ID),
			Title:             node.Title,
			EnglishTitle:      node.AlternativeTitles.English,
			JapaneseTitle:     node.AlternativeTitles.Japanese,
			AlternativeTitles: alternativeTitles,
			Synopsis:          node.Synopsis,
			Type:              node.MediaType,
			Status:            node.Status,
			Episodes:          node.NumEpisodes,
			Year:              node.StartSeason.Year,
			Season:            node.StartSeason.Season,
			Rating:            node.Mean,
			Genres:            genres,
			Studios:           studios,
			ImageURL:          node.MainPicture.Large,
		}

		animes = append(animes, anime)
	}

	return animes, nil
}

// GetAnimeDetails gets detailed information about an anime
func (t *MALTracker) GetAnimeDetails(ctx context.Context, id string) (*AnimeInfo, error) {
	q := url.Values{}
	q.Set("fields", "id,title,alternative_titles,main_picture,synopsis,mean,status,genres,media_type,num_episodes,start_season,studios,start_date,end_date")

	resp, err := t.apiRequest(ctx, "GET", "/anime/"+id, q, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get anime details: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get anime details: %s (%d)", string(body), resp.StatusCode)
	}

	var result struct {
		ID                int    `json:"id"`
		Title             string `json:"title"`
		AlternativeTitles struct {
			English  string   `json:"en"`
			Japanese string   `json:"ja"`
			Synonyms []string `json:"synonyms"`
		} `json:"alternative_titles"`
		MainPicture struct {
			Medium string `json:"medium"`
			Large  string `json:"large"`
		} `json:"main_picture"`
		Synopsis string  `json:"synopsis"`
		Mean     float64 `json:"mean"`
		Status   string  `json:"status"`
		Genres   []struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		} `json:"genres"`
		MediaType   string `json:"media_type"`
		NumEpisodes int    `json:"num_episodes"`
		StartSeason struct {
			Year   int    `json:"year"`
			Season string `json:"season"`
		} `json:"start_season"`
		Studios []struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		} `json:"studios"`
		StartDate string `json:"start_date"`
		EndDate   string `json:"end_date"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode anime details: %w", err)
	}

	genres := make([]string, 0, len(result.Genres))
	for _, g := range result.Genres {
		genres = append(genres, g.Name)
	}

	studios := make([]string, 0, len(result.Studios))
	for _, s := range result.Studios {
		studios = append(studios, s.Name)
	}

	var alternativeTitles []string
	if result.AlternativeTitles.English != "" {
		alternativeTitles = append(alternativeTitles, result.AlternativeTitles.English)
	}
	if result.AlternativeTitles.Japanese != "" {
		alternativeTitles = append(alternativeTitles, result.AlternativeTitles.Japanese)
	}
	alternativeTitles = append(alternativeTitles, result.AlternativeTitles.Synonyms...)

	var startDate, endDate time.Time
	if result.StartDate != "" {
		startDate, _ = time.Parse("2006-01-02", result.StartDate)
	}
	if result.EndDate != "" {
		endDate, _ = time.Parse("2006-01-02", result.EndDate)
	}

	anime := &AnimeInfo{
		ID:                strconv.Itoa(result.ID),
		Title:             result.Title,
		EnglishTitle:      result.AlternativeTitles.English,
		JapaneseTitle:     result.AlternativeTitles.Japanese,
		AlternativeTitles: alternativeTitles,
		Synopsis:          result.Synopsis,
		Type:              result.MediaType,
		Status:            result.Status,
		Episodes:          result.NumEpisodes,
		StartDate:         startDate,
		EndDate:           endDate,
		Year:              result.StartSeason.Year,
		Season:            result.StartSeason.Season,
		Rating:            result.Mean,
		Genres:            genres,
		Studios:           studios,
		ImageURL:          result.MainPicture.Large,
	}

	return anime, nil
}

// GetUserAnimeList gets the user's anime list
func (t *MALTracker) GetUserAnimeList(ctx context.Context) ([]UserAnimeEntry, error) {
	entries := []UserAnimeEntry{}
	offset := 0
	limit := 100
	hasNextPage := true

	for hasNextPage {
		list, nextOffset, err := t.getUserAnimeListPage(ctx, offset, limit)
		if err != nil {
			return nil, err
		}

		entries = append(entries, list...)

		if nextOffset > 0 {
			offset = nextOffset
		} else {
			hasNextPage = false
		}
	}

	return entries, nil
}

// getUserAnimeListPage gets a page of the user's anime list
func (t *MALTracker) getUserAnimeListPage(ctx context.Context, offset, limit int) ([]UserAnimeEntry, int, error) {
	q := url.Values{}
	q.Set("offset", strconv.Itoa(offset))
	q.Set("limit", strconv.Itoa(limit))
	q.Set("fields", "list_status,title,alternative_titles,main_picture,synopsis,mean,status,genres,media_type,num_episodes,start_season,studios,start_date,end_date")
	q.Set("nsfw", "true")

	resp, err := t.apiRequest(ctx, "GET", "/users/@me/animelist", q, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get user anime list: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, 0, fmt.Errorf("failed to get user anime list: %s (%d)", string(body), resp.StatusCode)
	}

	var result struct {
		Data []struct {
			Node struct {
				ID                int    `json:"id"`
				Title             string `json:"title"`
				AlternativeTitles struct {
					English  string   `json:"en"`
					Japanese string   `json:"ja"`
					Synonyms []string `json:"synonyms"`
				} `json:"alternative_titles"`
				MainPicture struct {
					Medium string `json:"medium"`
					Large  string `json:"large"`
				} `json:"main_picture"`
				Synopsis string  `json:"synopsis"`
				Mean     float64 `json:"mean"`
				Status   string  `json:"status"`
				Genres   []struct {
					ID   int    `json:"id"`
					Name string `json:"name"`
				} `json:"genres"`
				MediaType   string `json:"media_type"`
				NumEpisodes int    `json:"num_episodes"`
				StartSeason struct {
					Year   int    `json:"year"`
					Season string `json:"season"`
				} `json:"start_season"`
				Studios []struct {
					ID   int    `json:"id"`
					Name string `json:"name"`
				} `json:"studios"`
				StartDate string `json:"start_date"`
				EndDate   string `json:"end_date"`
			} `json:"node"`
			ListStatus struct {
				Status       string  `json:"status"`
				Score        float64 `json:"score"`
				NumEpisodes  int     `json:"num_episodes_watched"`
				StartDate    string  `json:"start_date"`
				FinishDate   string  `json:"finish_date"`
				Comments     string  `json:"comments"`
				UpdatedAt    string  `json:"updated_at"`
				IsRewatching bool    `json:"is_rewatching"`
				RewatchCount int     `json:"num_times_rewatched"`
				RewatchValue int     `json:"rewatch_value"`
			} `json:"list_status"`
		} `json:"data"`
		Paging struct {
			Next string `json:"next"`
		} `json:"paging"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, 0, fmt.Errorf("failed to decode user anime list: %w", err)
	}

	entries := make([]UserAnimeEntry, 0, len(result.Data))
	for _, item := range result.Data {
		node := item.Node
		status := item.ListStatus

		genres := make([]string, 0, len(node.Genres))
		for _, g := range node.Genres {
			genres = append(genres, g.Name)
		}

		studios := make([]string, 0, len(node.Studios))
		for _, s := range node.Studios {
			studios = append(studios, s.Name)
		}

		var alternativeTitles []string
		if node.AlternativeTitles.English != "" {
			alternativeTitles = append(alternativeTitles, node.AlternativeTitles.English)
		}
		if node.AlternativeTitles.Japanese != "" {
			alternativeTitles = append(alternativeTitles, node.AlternativeTitles.Japanese)
		}
		alternativeTitles = append(alternativeTitles, node.AlternativeTitles.Synonyms...)

		var startDate, endDate, listStartDate, listFinishDate, updatedAt time.Time
		if node.StartDate != "" {
			startDate, _ = time.Parse("2006-01-02", node.StartDate)
		}
		if node.EndDate != "" {
			endDate, _ = time.Parse("2006-01-02", node.EndDate)
		}
		if status.StartDate != "" {
			listStartDate, _ = time.Parse("2006-01-02", status.StartDate)
		}
		if status.FinishDate != "" {
			listFinishDate, _ = time.Parse("2006-01-02", status.FinishDate)
		}
		if status.UpdatedAt != "" {
			updatedAt, _ = time.Parse(time.RFC3339, status.UpdatedAt)
		}

		// Map MAL status to our status enum
		animeStatus := StatusWatching
		switch status.Status {
		case "watching":
			animeStatus = StatusWatching
		case "completed":
			animeStatus = StatusCompleted
		case "on_hold":
			animeStatus = StatusOnHold
		case "dropped":
			animeStatus = StatusDropped
		case "plan_to_watch":
			animeStatus = StatusPlanToWatch
		}

		entry := UserAnimeEntry{
			AnimeInfo: AnimeInfo{
				ID:                strconv.Itoa(node.ID),
				Title:             node.Title,
				EnglishTitle:      node.AlternativeTitles.English,
				JapaneseTitle:     node.AlternativeTitles.Japanese,
				AlternativeTitles: alternativeTitles,
				Synopsis:          node.Synopsis,
				Type:              node.MediaType,
				Status:            node.Status,
				Episodes:          node.NumEpisodes,
				StartDate:         startDate,
				EndDate:           endDate,
				Year:              node.StartSeason.Year,
				Season:            node.StartSeason.Season,
				Rating:            node.Mean,
				Genres:            genres,
				Studios:           studios,
				ImageURL:          node.MainPicture.Large,
			},
			Status:      animeStatus,
			Score:       status.Score,
			Progress:    float64(status.NumEpisodes),
			StartDate:   listStartDate,
			EndDate:     listFinishDate,
			Notes:       status.Comments,
			LastUpdated: updatedAt,
		}

		entries = append(entries, entry)
	}

	// Extract next offset from paging URL if available
	nextOffset := 0
	if result.Paging.Next != "" {
		u, err := url.Parse(result.Paging.Next)
		if err == nil {
			if offsetStr := u.Query().Get("offset"); offsetStr != "" {
				nextOffset, _ = strconv.Atoi(offsetStr)
			}
		}
	}

	return entries, nextOffset, nil
}

// UpdateAnimeStatus updates the watch status of an anime
func (t *MALTracker) UpdateAnimeStatus(ctx context.Context, id string, status Status, episode float64, score float64) error {
	data := url.Values{}

	// Map our status to MAL status
	malStatus := "watching"
	switch status {
	case StatusWatching:
		malStatus = "watching"
	case StatusCompleted:
		malStatus = "completed"
	case StatusOnHold:
		malStatus = "on_hold"
	case StatusDropped:
		malStatus = "dropped"
	case StatusPlanToWatch:
		malStatus = "plan_to_watch"
	}

	data.Set("status", malStatus)

	if episode > 0 {
		data.Set("num_watched_episodes", strconv.Itoa(int(episode)))
	}

	if score > 0 {
		data.Set("score", strconv.Itoa(int(score)))
	}

	req, err := http.NewRequestWithContext(ctx, "PATCH", fmt.Sprintf("%s/anime/%s/my_list_status", malAPIBaseURL, id), strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create update request: %w", err)
	}

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Authorization", fmt.Sprintf("%s %s", t.token.TokenType, t.token.AccessToken))

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to update anime status: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to update anime status: %s (%d)", string(body), resp.StatusCode)
	}

	return nil
}

// SyncFromRemote synchronizes the local database with MAL
func (t *MALTracker) SyncFromRemote(ctx context.Context, db *database.DB) (SyncStats, error) {
	stats := SyncStats{
		Details: []string{},
	}

	// Get user's anime list from MAL
	entries, err := t.GetUserAnimeList(ctx)
	if err != nil {
		return stats, fmt.Errorf("failed to get user anime list: %w", err)
	}

	for _, entry := range entries {
		// Check if anime exists in database
		anime, err := db.GetAnimeByExternalID(entry.ID, t.Name())
		if err != nil && err != sql.ErrNoRows {
			stats.Errors++
			stats.Details = append(stats.Details, fmt.Sprintf("Error checking anime %s: %v", entry.Title, err))
			continue
		}

		if anime == nil {
			// Anime doesn't exist, add it
			animeData := &database.Anime{
				Title:             entry.Title,
				OriginalTitle:     entry.JapaneseTitle,
				AlternativeTitles: entry.AlternativeTitles,
				Description:       entry.Synopsis,
				TotalEpisodes:     entry.Episodes,
				Type:              entry.Type,
				Year:              entry.Year,
				Season:            entry.Season,
				Status:            string(entry.Status),
				Genres:            entry.Genres,
				ThumbnailURL:      entry.ImageURL,
			}

			if err := db.AddAnime(animeData); err != nil {
				stats.Errors++
				stats.Details = append(stats.Details, fmt.Sprintf("Failed to add anime %s: %v", entry.Title, err))
				continue
			}

			// Add tracking information
			tracking := &database.AnimeTracking{
				AnimeID:        animeData.ID,
				Tracker:        t.Name(),
				TrackerID:      entry.ID,
				Status:         string(entry.Status),
				Score:          entry.Score,
				CurrentEpisode: entry.Progress,
				TotalEpisodes:  entry.Episodes,
				LastUpdated:    entry.LastUpdated,
			}

			if err := db.AddAnimeTracking(tracking); err != nil {
				stats.Errors++
				stats.Details = append(stats.Details, fmt.Sprintf("Failed to add tracking for %s: %v", entry.Title, err))
				continue
			}

			stats.Added++
			stats.Details = append(stats.Details, fmt.Sprintf("Added anime: %s", entry.Title))
		} else {
			// Anime exists, update tracking
			tracking, err := db.GetAnimeTracking(anime.ID, t.Name())
			if err != nil && err != sql.ErrNoRows {
				stats.Errors++
				stats.Details = append(stats.Details, fmt.Sprintf("Error checking tracking for %s: %v", entry.Title, err))
				continue
			}

			if tracking == nil {
				// Tracking doesn't exist, add it
				tracking := &database.AnimeTracking{
					AnimeID:        anime.ID,
					Tracker:        t.Name(),
					TrackerID:      entry.ID,
					Status:         string(entry.Status),
					Score:          entry.Score,
					CurrentEpisode: entry.Progress,
					TotalEpisodes:  entry.Episodes,
					LastUpdated:    entry.LastUpdated,
				}

				if err := db.AddAnimeTracking(tracking); err != nil {
					stats.Errors++
					stats.Details = append(stats.Details, fmt.Sprintf("Failed to add tracking for %s: %v", entry.Title, err))
					continue
				}

				stats.Added++
				stats.Details = append(stats.Details, fmt.Sprintf("Added tracking for: %s", entry.Title))
			} else {
				// Tracking exists, update if remote is newer
				if entry.LastUpdated.After(tracking.LastUpdated) {
					tracking.Status = string(entry.Status)
					tracking.Score = entry.Score
					tracking.CurrentEpisode = entry.Progress
					tracking.TotalEpisodes = entry.Episodes
					tracking.LastUpdated = entry.LastUpdated

					if err := db.UpdateAnimeTrackingObject(tracking); err != nil {
						stats.Errors++
						stats.Details = append(stats.Details, fmt.Sprintf("Failed to update tracking for %s: %v", entry.Title, err))
						continue
					}

					stats.Updated++
					stats.Details = append(stats.Details, fmt.Sprintf("Updated tracking for: %s", entry.Title))
				} else {
					stats.Skipped++
				}
			}
		}
	}

	// Update last sync time
	if err := db.SetConfig("mal_last_sync", time.Now().Format(time.RFC3339)); err != nil {
		return stats, fmt.Errorf("failed to update last sync time: %w", err)
	}

	return stats, nil
}

// SyncToRemote synchronizes MAL with the local database
func (t *MALTracker) SyncToRemote(ctx context.Context, db *database.DB) (SyncStats, error) {
	stats := SyncStats{
		Details: []string{},
	}

	// Get all anime with MAL tracking that have been updated since last sync
	lastSyncStr, err := db.GetConfig("mal_last_sync")
	if err != nil && err != sql.ErrNoRows {
		return stats, fmt.Errorf("failed to get last sync time: %w", err)
	}

	var lastSync time.Time
	if lastSyncStr != "" {
		lastSync, err = time.Parse(time.RFC3339, lastSyncStr)
		if err != nil {
			return stats, fmt.Errorf("failed to parse last sync time: %w", err)
		}
	}

	// Get all tracking entries for MAL
	trackings, err := db.GetAllAnimeTrackingByTracker(t.Name())
	if err != nil {
		return stats, fmt.Errorf("failed to get MAL tracking entries: %w", err)
	}

	for _, tracking := range trackings {
		// Skip if tracking hasn't been updated since last sync
		if !lastSync.IsZero() && !tracking.LastUpdated.After(lastSync) {
			stats.Skipped++
			continue
		}

		// Map status string to enum
		status := StatusWatching
		switch tracking.Status {
		case "watching":
			status = StatusWatching
		case "completed":
			status = StatusCompleted
		case "on_hold":
			status = StatusOnHold
		case "dropped":
			status = StatusDropped
		case "plan_to_watch":
			status = StatusPlanToWatch
		}

		// Update MAL
		if err := t.UpdateAnimeStatus(ctx, tracking.TrackerID, status, tracking.CurrentEpisode, tracking.Score); err != nil {
			stats.Errors++
			stats.Details = append(stats.Details, fmt.Sprintf("Failed to update anime %s on MAL: %v", tracking.TrackerID, err))
			continue
		}

		stats.Updated++
		stats.Details = append(stats.Details, fmt.Sprintf("Updated anime %s on MAL", tracking.TrackerID))
	}

	// Update last sync time
	if err := db.SetConfig("mal_last_sync", time.Now().Format(time.RFC3339)); err != nil {
		return stats, fmt.Errorf("failed to update last sync time: %w", err)
	}

	return stats, nil
}
