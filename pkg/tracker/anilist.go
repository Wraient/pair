package tracker

import (
	"bytes"
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
	anilistAPIURL        = "https://graphql.anilist.co"
	anilistOAuthURL      = "https://anilist.co/api/v2/oauth"
	anilistClientID      = "27391"
	anilistClientSecret  = "8EYStp7HAAuR41zPQgKf1boP7WOWWFsOKpg4pCuZ"
	anilistTokenFilename = "anilist_token.json"
	anilistRedirectURI   = "http://localhost:8000/oauth/callback"
	anilistServerPort    = 8000
)

// AnilistToken represents the OAuth token response from Anilist
type AnilistToken struct {
	AccessToken  string    `json:"access_token"`
	TokenType    string    `json:"token_type"`
	ExpiresIn    int       `json:"expires_in"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// AnilistTracker implements the Tracker interface for Anilist
type AnilistTracker struct {
	token      *AnilistToken
	tokenPath  string
	httpClient *http.Client
}

// NewAnilistTracker creates a new AnilistTracker
func NewAnilistTracker(configDir string) *AnilistTracker {
	if err := os.MkdirAll(configDir, 0755); err != nil {
		fmt.Printf("Failed to create config directory: %v\n", err)
	}

	return &AnilistTracker{
		tokenPath:  filepath.Join(configDir, anilistTokenFilename),
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// Name returns the name of the tracker
func (t *AnilistTracker) Name() string {
	return "anilist"
}

// IsAuthenticated checks if the user is authenticated with the tracker
func (t *AnilistTracker) IsAuthenticated() bool {
	if t.token == nil {
		if err := t.loadToken(); err != nil {
			return false
		}
	}
	return t.token != nil && t.token.AccessToken != "" && time.Now().Before(t.token.ExpiresAt)
}

// loadToken loads the token from the token file
func (t *AnilistTracker) loadToken() error {
	data, err := os.ReadFile(t.tokenPath)
	if err != nil {
		return fmt.Errorf("failed to read token file: %w", err)
	}

	var token AnilistToken
	if err := json.Unmarshal(data, &token); err != nil {
		return fmt.Errorf("failed to parse token file: %w", err)
	}

	t.token = &token
	return nil
}

// saveToken saves the token to the token file
func (t *AnilistTracker) saveToken() error {
	data, err := json.Marshal(t.token)
	if err != nil {
		return fmt.Errorf("failed to marshal token: %w", err)
	}

	return os.WriteFile(t.tokenPath, data, 0600)
}

// refreshToken refreshes the access token using the refresh token
func (t *AnilistTracker) refreshToken() error {
	if t.token == nil || t.token.RefreshToken == "" {
		return fmt.Errorf("no refresh token available")
	}

	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("client_id", anilistClientID)
	data.Set("client_secret", anilistClientSecret)
	data.Set("refresh_token", t.token.RefreshToken)

	req, err := http.NewRequest("POST", anilistOAuthURL+"/token", strings.NewReader(data.Encode()))
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

	var newToken AnilistToken
	if err := json.NewDecoder(resp.Body).Decode(&newToken); err != nil {
		return fmt.Errorf("failed to decode token response: %w", err)
	}

	newToken.ExpiresAt = time.Now().Add(time.Duration(newToken.ExpiresIn) * time.Second)
	t.token = &newToken

	return t.saveToken()
}

// Authenticate authenticates the user with Anilist
func (t *AnilistTracker) Authenticate(ctx context.Context) error {
	// Try to load existing token first
	if err := t.loadToken(); err == nil && t.IsAuthenticated() {
		return nil
	}

	// Try to refresh token if available
	if t.token != nil && t.token.RefreshToken != "" {
		if err := t.refreshToken(); err == nil {
			return nil
		}
	}

	// Start local server to handle OAuth callback
	callbackCh := make(chan string, 1)
	errCh := make(chan error, 1)
	srv := &http.Server{
		Addr: fmt.Sprintf(":%d", anilistServerPort),
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/oauth/callback" {
				http.Error(w, "Not found", http.StatusNotFound)
				return
			}

			code := r.URL.Query().Get("code")
			if code == "" {
				errCh <- fmt.Errorf("no code in callback")
				http.Error(w, "No code received", http.StatusBadRequest)
				return
			}

			w.Header().Set("Content-Type", "text/html")
			fmt.Fprintf(w, "<html><body><h1>Authentication Successful!</h1><p>You can close this window now.</p><script>window.close()</script></body></html>")
			callbackCh <- code
		}),
	}

	// Start server in background
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()
	defer srv.Shutdown(ctx)

	// Open browser for authentication
	authURL := fmt.Sprintf("%s/authorize?client_id=%s&redirect_uri=%s&response_type=code",
		anilistOAuthURL,
		anilistClientID,
		url.QueryEscape(anilistRedirectURI))

	if err := browser.OpenURL(authURL); err != nil {
		return fmt.Errorf("failed to open browser: %w", err)
	}

	// Wait for callback
	var code string
	select {
	case code = <-callbackCh:
	case err := <-errCh:
		return fmt.Errorf("authentication failed: %w", err)
	case <-ctx.Done():
		return ctx.Err()
	}

	// Exchange code for token
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("client_id", anilistClientID)
	data.Set("client_secret", anilistClientSecret)
	data.Set("redirect_uri", anilistRedirectURI)
	data.Set("code", code)

	req, err := http.NewRequest("POST", anilistOAuthURL+"/token", strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create token request: %w", err)
	}

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	resp, err := t.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to exchange code for token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("token exchange failed: %s (%d)", string(body), resp.StatusCode)
	}

	var token AnilistToken
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return fmt.Errorf("failed to decode token response: %w", err)
	}

	token.ExpiresAt = time.Now().Add(time.Duration(token.ExpiresIn) * time.Second)
	t.token = &token

	return t.saveToken()
}

// graphqlRequest sends a GraphQL request to the Anilist API
func (t *AnilistTracker) graphqlRequest(ctx context.Context, query string, variables map[string]interface{}) ([]byte, error) {
	if !t.IsAuthenticated() {
		if err := t.Authenticate(ctx); err != nil {
			return nil, fmt.Errorf("authentication required: %w", err)
		}
	}

	reqBody := struct {
		Query     string                 `json:"query"`
		Variables map[string]interface{} `json:"variables,omitempty"`
	}{
		Query:     query,
		Variables: variables,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", anilistAPIURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+t.token.AccessToken)

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Handle unauthorized response by refreshing token and retrying
	if resp.StatusCode == http.StatusUnauthorized {
		if err := t.refreshToken(); err != nil {
			return nil, fmt.Errorf("token refresh failed: %w", err)
		}

		// Retry request with new token
		req.Header.Set("Authorization", "Bearer "+t.token.AccessToken)
		resp, err = t.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("retry request failed: %w", err)
		}
		defer resp.Body.Close()
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check for GraphQL errors
	var errorResp struct {
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(body, &errorResp); err == nil && len(errorResp.Errors) > 0 {
		var messages []string
		for _, e := range errorResp.Errors {
			messages = append(messages, e.Message)
		}
		return nil, fmt.Errorf("GraphQL errors: %s", strings.Join(messages, "; "))
	}

	return body, nil
}

// SearchAnime searches for anime on Anilist
func (t *AnilistTracker) SearchAnime(ctx context.Context, query string, limit int) ([]AnimeInfo, error) {
	gqlQuery := `
	query ($search: String, $perPage: Int) {
		Page(page: 1, perPage: $perPage) {
			media(search: $search, type: ANIME) {
				id
				title {
					romaji
					english
					native
					userPreferred
				}
				synonyms
				description
				format
				status
				episodes
				duration
				genres
				studios {
					nodes {
						name
					}
				}
				seasonYear
				season
				averageScore
				coverImage {
					large
				}
				startDate {
					year
					month
					day
				}
				endDate {
					year
					month
					day
				}
			}
		}
	}
	`

	variables := map[string]interface{}{
		"search":  query,
		"perPage": limit,
	}

	resp, err := t.graphqlRequest(ctx, gqlQuery, variables)
	if err != nil {
		return nil, fmt.Errorf("failed to search anime: %w", err)
	}

	var result struct {
		Data struct {
			Page struct {
				Media []struct {
					ID    int `json:"id"`
					Title struct {
						Romaji        string `json:"romaji"`
						English       string `json:"english"`
						Native        string `json:"native"`
						UserPreferred string `json:"userPreferred"`
					} `json:"title"`
					Synonyms    []string `json:"synonyms"`
					Description string   `json:"description"`
					Format      string   `json:"format"`
					Status      string   `json:"status"`
					Episodes    int      `json:"episodes"`
					Duration    int      `json:"duration"`
					Genres      []string `json:"genres"`
					Studios     struct {
						Nodes []struct {
							Name string `json:"name"`
						} `json:"nodes"`
					} `json:"studios"`
					SeasonYear   int     `json:"seasonYear"`
					Season       string  `json:"season"`
					AverageScore float64 `json:"averageScore"`
					CoverImage   struct {
						Large string `json:"large"`
					} `json:"coverImage"`
					StartDate struct {
						Year  int `json:"year"`
						Month int `json:"month"`
						Day   int `json:"day"`
					} `json:"startDate"`
					EndDate struct {
						Year  int `json:"year"`
						Month int `json:"month"`
						Day   int `json:"day"`
					} `json:"endDate"`
				} `json:"media"`
			} `json:"Page"`
		} `json:"data"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse search results: %w", err)
	}

	animes := make([]AnimeInfo, 0, len(result.Data.Page.Media))
	for _, media := range result.Data.Page.Media {
		studios := make([]string, 0, len(media.Studios.Nodes))
		for _, studio := range media.Studios.Nodes {
			studios = append(studios, studio.Name)
		}

		var startDate, endDate time.Time
		if media.StartDate.Year > 0 {
			startDate = time.Date(
				media.StartDate.Year,
				time.Month(media.StartDate.Month),
				media.StartDate.Day,
				0, 0, 0, 0, time.UTC,
			)
		}
		if media.EndDate.Year > 0 {
			endDate = time.Date(
				media.EndDate.Year,
				time.Month(media.EndDate.Month),
				media.EndDate.Day,
				0, 0, 0, 0, time.UTC,
			)
		}

		// Create alternative titles list
		alternativeTitles := make([]string, 0)
		if media.Title.English != "" {
			alternativeTitles = append(alternativeTitles, media.Title.English)
		}
		if media.Title.Romaji != "" && media.Title.Romaji != media.Title.English {
			alternativeTitles = append(alternativeTitles, media.Title.Romaji)
		}
		if media.Title.Native != "" {
			alternativeTitles = append(alternativeTitles, media.Title.Native)
		}
		alternativeTitles = append(alternativeTitles, media.Synonyms...)

		anime := AnimeInfo{
			ID:                strconv.Itoa(media.ID),
			Title:             media.Title.UserPreferred,
			EnglishTitle:      media.Title.English,
			JapaneseTitle:     media.Title.Native,
			AlternativeTitles: alternativeTitles,
			Synopsis:          media.Description,
			Type:              media.Format,
			Status:            media.Status,
			Episodes:          media.Episodes,
			StartDate:         startDate,
			EndDate:           endDate,
			Year:              media.SeasonYear,
			Season:            strings.ToLower(media.Season),
			Rating:            media.AverageScore / 10.0, // Convert to 10-point scale
			Genres:            media.Genres,
			Studios:           studios,
			ImageURL:          media.CoverImage.Large,
		}

		animes = append(animes, anime)
	}

	return animes, nil
}

// GetAnimeDetails gets detailed information about an anime
func (t *AnilistTracker) GetAnimeDetails(ctx context.Context, id string) (*AnimeInfo, error) {
	gqlQuery := `
	query ($id: Int) {
		Media(id: $id, type: ANIME) {
			id
			title {
				romaji
				english
				native
				userPreferred
			}
			synonyms
			description
			format
			status
			episodes
			duration
			genres
			studios {
				nodes {
					name
				}
			}
			seasonYear
			season
			averageScore
			coverImage {
				large
			}
			startDate {
				year
				month
				day
			}
			endDate {
				year
				month
				day
			}
		}
	}
	`

	mediaID, err := strconv.Atoi(id)
	if err != nil {
		return nil, fmt.Errorf("invalid ID: %w", err)
	}

	variables := map[string]interface{}{
		"id": mediaID,
	}

	resp, err := t.graphqlRequest(ctx, gqlQuery, variables)
	if err != nil {
		return nil, fmt.Errorf("failed to get anime details: %w", err)
	}

	var result struct {
		Data struct {
			Media struct {
				ID    int `json:"id"`
				Title struct {
					Romaji        string `json:"romaji"`
					English       string `json:"english"`
					Native        string `json:"native"`
					UserPreferred string `json:"userPreferred"`
				} `json:"title"`
				Synonyms    []string `json:"synonyms"`
				Description string   `json:"description"`
				Format      string   `json:"format"`
				Status      string   `json:"status"`
				Episodes    int      `json:"episodes"`
				Duration    int      `json:"duration"`
				Genres      []string `json:"genres"`
				Studios     struct {
					Nodes []struct {
						Name string `json:"name"`
					} `json:"nodes"`
				} `json:"studios"`
				SeasonYear   int     `json:"seasonYear"`
				Season       string  `json:"season"`
				AverageScore float64 `json:"averageScore"`
				CoverImage   struct {
					Large string `json:"large"`
				} `json:"coverImage"`
				StartDate struct {
					Year  int `json:"year"`
					Month int `json:"month"`
					Day   int `json:"day"`
				} `json:"startDate"`
				EndDate struct {
					Year  int `json:"year"`
					Month int `json:"month"`
					Day   int `json:"day"`
				} `json:"endDate"`
			} `json:"Media"`
		} `json:"data"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse anime details: %w", err)
	}

	media := result.Data.Media

	studios := make([]string, 0, len(media.Studios.Nodes))
	for _, studio := range media.Studios.Nodes {
		studios = append(studios, studio.Name)
	}

	var startDate, endDate time.Time
	if media.StartDate.Year > 0 {
		startDate = time.Date(
			media.StartDate.Year,
			time.Month(media.StartDate.Month),
			media.StartDate.Day,
			0, 0, 0, 0, time.UTC,
		)
	}
	if media.EndDate.Year > 0 {
		endDate = time.Date(
			media.EndDate.Year,
			time.Month(media.EndDate.Month),
			media.EndDate.Day,
			0, 0, 0, 0, time.UTC,
		)
	}

	// Create alternative titles list
	alternativeTitles := make([]string, 0)
	if media.Title.English != "" {
		alternativeTitles = append(alternativeTitles, media.Title.English)
	}
	if media.Title.Romaji != "" && media.Title.Romaji != media.Title.English {
		alternativeTitles = append(alternativeTitles, media.Title.Romaji)
	}
	if media.Title.Native != "" {
		alternativeTitles = append(alternativeTitles, media.Title.Native)
	}
	alternativeTitles = append(alternativeTitles, media.Synonyms...)

	anime := &AnimeInfo{
		ID:                strconv.Itoa(media.ID),
		Title:             media.Title.UserPreferred,
		EnglishTitle:      media.Title.English,
		JapaneseTitle:     media.Title.Native,
		AlternativeTitles: alternativeTitles,
		Synopsis:          media.Description,
		Type:              media.Format,
		Status:            media.Status,
		Episodes:          media.Episodes,
		StartDate:         startDate,
		EndDate:           endDate,
		Year:              media.SeasonYear,
		Season:            strings.ToLower(media.Season),
		Rating:            media.AverageScore / 10.0, // Convert to 10-point scale
		Genres:            media.Genres,
		Studios:           studios,
		ImageURL:          media.CoverImage.Large,
	}

	return anime, nil
}

// getCurrentUser gets the current user's information
func (t *AnilistTracker) getCurrentUser(ctx context.Context) (int, error) {
	query := `
	query {
		Viewer {
			id
			name
		}
	}`

	resp, err := t.graphqlRequest(ctx, query, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to get current user: %w", err)
	}

	var result struct {
		Data struct {
			Viewer struct {
				ID   int    `json:"id"`
				Name string `json:"name"`
			} `json:"Viewer"`
		} `json:"data"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return 0, fmt.Errorf("failed to parse response: %w", err)
	}

	return result.Data.Viewer.ID, nil
}

// GetUserAnimeList gets the user's anime list
func (t *AnilistTracker) GetUserAnimeList(ctx context.Context) ([]UserAnimeEntry, error) {
	// Get current user's ID
	userID, err := t.getCurrentUser(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get user ID: %w", err)
	}

	query := `
	query ($userId: Int) {
		MediaListCollection(userId: $userId, type: ANIME) {
			lists {
				entries {
					media {
						id
						title {
							userPreferred
							english
							native
							romaji
						}
						coverImage {
							large
						}
						format
						episodes
						duration
						status
						season
						seasonYear
						genres
						averageScore
						description
						studios {
							nodes {
								name
							}
						}
					}
					score
					progress
					status
					startedAt {
						year
						month
						day
					}
					completedAt {
						year
						month
						day
					}
					updatedAt
					notes
				}
			}
		}
	}`

	variables := map[string]interface{}{
		"userId": userID,
	}

	resp, err := t.graphqlRequest(ctx, query, variables)
	if err != nil {
		return nil, fmt.Errorf("failed to get user anime list: %w", err)
	}

	var result struct {
		Data struct {
			MediaListCollection struct {
				Lists []struct {
					Entries []struct {
						Media struct {
							ID    int `json:"id"`
							Title struct {
								UserPreferred string `json:"userPreferred"`
								English       string `json:"english"`
								Native        string `json:"native"`
								Romaji        string `json:"romaji"`
							} `json:"title"`
							CoverImage struct {
								Large string `json:"large"`
							} `json:"coverImage"`
							Format       string   `json:"format"`
							Episodes     int      `json:"episodes"`
							Duration     int      `json:"duration"`
							Status       string   `json:"status"`
							Season       string   `json:"season"`
							SeasonYear   int      `json:"seasonYear"`
							Genres       []string `json:"genres"`
							AverageScore float64  `json:"averageScore"`
							Description  string   `json:"description"`
							Studios      struct {
								Nodes []struct {
									Name string `json:"name"`
								} `json:"nodes"`
							} `json:"studios"`
						} `json:"media"`
						Score     float64 `json:"score"`
						Progress  int     `json:"progress"`
						Status    string  `json:"status"`
						StartedAt struct {
							Year  int `json:"year"`
							Month int `json:"month"`
							Day   int `json:"day"`
						} `json:"startedAt"`
						CompletedAt struct {
							Year  int `json:"year"`
							Month int `json:"month"`
							Day   int `json:"day"`
						} `json:"completedAt"`
						UpdatedAt int64  `json:"updatedAt"`
						Notes     string `json:"notes"`
					} `json:"entries"`
				} `json:"lists"`
			} `json:"MediaListCollection"`
		} `json:"data"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	entries := make([]UserAnimeEntry, 0)
	for _, list := range result.Data.MediaListCollection.Lists {
		for _, item := range list.Entries {
			media := item.Media

			// Convert dates
			var startDate, endDate time.Time
			if item.StartedAt.Year > 0 {
				startDate = time.Date(item.StartedAt.Year, time.Month(item.StartedAt.Month), item.StartedAt.Day, 0, 0, 0, 0, time.UTC)
			}
			if item.CompletedAt.Year > 0 {
				endDate = time.Date(item.CompletedAt.Year, time.Month(item.CompletedAt.Month), item.CompletedAt.Day, 0, 0, 0, 0, time.UTC)
			}

			// Get studios
			studios := make([]string, 0, len(media.Studios.Nodes))
			for _, studio := range media.Studios.Nodes {
				studios = append(studios, studio.Name)
			}

			// Map status
			status := StatusWatching
			switch strings.ToLower(item.Status) {
			case "current":
				status = StatusWatching
			case "completed":
				status = StatusCompleted
			case "paused":
				status = StatusOnHold
			case "dropped":
				status = StatusDropped
			case "planning":
				status = StatusPlanToWatch
			}

			entry := UserAnimeEntry{
				AnimeInfo: AnimeInfo{
					ID:            strconv.Itoa(media.ID),
					Title:         media.Title.UserPreferred,
					EnglishTitle:  media.Title.English,
					JapaneseTitle: media.Title.Native,
					Synopsis:      media.Description,
					Type:          media.Format,
					Episodes:      media.Episodes,
					Status:        media.Status,
					Year:          media.SeasonYear,
					Season:        strings.ToLower(media.Season),
					Rating:        media.AverageScore / 10.0,
					Genres:        media.Genres,
					Studios:       studios,
					ImageURL:      media.CoverImage.Large,
				},
				Status:      status,
				Score:       item.Score,
				Progress:    float64(item.Progress),
				StartDate:   startDate,
				EndDate:     endDate,
				Notes:       item.Notes,
				LastUpdated: time.Unix(item.UpdatedAt, 0),
			}

			entries = append(entries, entry)
		}
	}

	return entries, nil
}

// UpdateAnimeStatus updates the watch status of an anime
func (t *AnilistTracker) UpdateAnimeStatus(ctx context.Context, id string, status Status, episode float64, score float64) error {
	mediaID, err := strconv.Atoi(id)
	if err != nil {
		return fmt.Errorf("invalid ID: %w", err)
	}

	// Map our status to Anilist status
	anilistStatus := "CURRENT"
	switch status {
	case StatusWatching:
		anilistStatus = "CURRENT"
	case StatusCompleted:
		anilistStatus = "COMPLETED"
	case StatusOnHold:
		anilistStatus = "PAUSED"
	case StatusDropped:
		anilistStatus = "DROPPED"
	case StatusPlanToWatch:
		anilistStatus = "PLANNING"
	}

	gqlQuery := `
	mutation ($mediaId: Int, $status: MediaListStatus, $progress: Int, $score: Float) {
		SaveMediaListEntry(mediaId: $mediaId, status: $status, progress: $progress, score: $score) {
			id
			status
			progress
			score
		}
	}
	`

	variables := map[string]interface{}{
		"mediaId": mediaID,
		"status":  anilistStatus,
	}

	if episode > 0 {
		variables["progress"] = int(episode)
	}

	if score > 0 {
		// Anilist uses a 10-point scale
		variables["score"] = score
	}

	_, err = t.graphqlRequest(ctx, gqlQuery, variables)
	if err != nil {
		return fmt.Errorf("failed to update anime status: %w", err)
	}

	return nil
}

// SyncFromRemote synchronizes the local database with Anilist
func (t *AnilistTracker) SyncFromRemote(ctx context.Context, db *database.DB) (SyncStats, error) {
	stats := SyncStats{
		Details: []string{},
	}

	// Get user's anime list from Anilist
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
	if err := db.SetConfig("anilist_last_sync", time.Now().Format(time.RFC3339)); err != nil {
		return stats, fmt.Errorf("failed to update last sync time: %w", err)
	}

	return stats, nil
}

// SyncToRemote synchronizes Anilist with the local database
func (t *AnilistTracker) SyncToRemote(ctx context.Context, db *database.DB) (SyncStats, error) {
	stats := SyncStats{
		Details: []string{},
	}

	// Get last sync time
	lastSyncStr, err := db.GetConfig("anilist_last_sync")
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

	// Get all tracking entries for Anilist
	trackings, err := db.GetAllAnimeTrackingByTracker(t.Name())
	if err != nil {
		return stats, fmt.Errorf("failed to get Anilist tracking entries: %w", err)
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

		// Update Anilist
		if err := t.UpdateAnimeStatus(ctx, tracking.TrackerID, status, tracking.CurrentEpisode, tracking.Score); err != nil {
			stats.Errors++
			stats.Details = append(stats.Details, fmt.Sprintf("Failed to update anime %s on Anilist: %v", tracking.TrackerID, err))
			continue
		}

		stats.Updated++
		stats.Details = append(stats.Details, fmt.Sprintf("Updated anime %s on Anilist", tracking.TrackerID))
	}

	// Update last sync time
	if err := db.SetConfig("anilist_last_sync", time.Now().Format(time.RFC3339)); err != nil {
		return stats, fmt.Errorf("failed to update last sync time: %w", err)
	}

	return stats, nil
}
