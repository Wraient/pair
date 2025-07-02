package ui

import (
	"fmt"
	"strings"

	"github.com/wraient/pair/pkg/tracker"
)

// ShowAnimeSearchResults displays search results in a CLI menu and returns the selected anime's ID
func ShowAnimeSearchResults(results []tracker.AnimeInfo) (string, error) {
	if len(results) == 0 {
		return "", fmt.Errorf("no results found")
	}

	items := make([]Pair, len(results))
	for i, anime := range results {
		// Create a display string with title and additional info
		displayInfo := []string{anime.Title}
		if anime.Year > 0 {
			displayInfo = append(displayInfo, fmt.Sprintf("(%d)", anime.Year))
		}
		if anime.Type != "" {
			displayInfo = append(displayInfo, anime.Type)
		}
		if anime.Episodes > 0 {
			displayInfo = append(displayInfo, fmt.Sprintf("%d eps", anime.Episodes))
		}

		items[i] = Pair{
			Label: strings.Join(displayInfo, " - "),
			Value: anime.ID,
		}
	}

	selectedID, err := ShowCLIMenu(List, items)
	if err != nil {
		return "", fmt.Errorf("menu error: %w", err)
	}

	return selectedID, nil
}

// ShowAnimeStatusSelection displays a menu to select anime status
func ShowAnimeStatusSelection() (tracker.Status, error) {
	items := []Pair{
		{Label: "Watching", Value: string(tracker.StatusWatching)},
		{Label: "Completed", Value: string(tracker.StatusCompleted)},
		{Label: "On Hold", Value: string(tracker.StatusOnHold)},
		{Label: "Dropped", Value: string(tracker.StatusDropped)},
		{Label: "Plan to Watch", Value: string(tracker.StatusPlanToWatch)},
	}

	status, err := ShowCLIMenu(List, items)
	if err != nil {
		return "", fmt.Errorf("menu error: %w", err)
	}

	return tracker.Status(status), nil
}

// ShowAnimeScoreSelection displays a menu to select anime score (0-10)
func ShowAnimeScoreSelection() (float64, error) {
	items := []Pair{
		{Label: "10 - Masterpiece", Value: "10"},
		{Label: "9 - Great", Value: "9"},
		{Label: "8 - Very Good", Value: "8"},
		{Label: "7 - Good", Value: "7"},
		{Label: "6 - Fine", Value: "6"},
		{Label: "5 - Average", Value: "5"},
		{Label: "4 - Bad", Value: "4"},
		{Label: "3 - Very Bad", Value: "3"},
		{Label: "2 - Horrible", Value: "2"},
		{Label: "1 - Appalling", Value: "1"},
		{Label: "0 - No Score", Value: "0"},
	}

	scoreStr, err := ShowCLIMenu(List, items)
	if err != nil {
		return 0, fmt.Errorf("menu error: %w", err)
	}

	// Convert score string to float64
	var score float64
	_, err = fmt.Sscanf(scoreStr, "%f", &score)
	if err != nil {
		return 0, fmt.Errorf("invalid score: %w", err)
	}

	return score, nil
}

// ShowAnimeList displays the user's anime list and returns the selected anime's ID
func ShowAnimeList(entries []tracker.UserAnimeEntry) (string, error) {
	if len(entries) == 0 {
		return "", fmt.Errorf("no entries found")
	}

	items := make([]Pair, len(entries))
	for i, entry := range entries {
		// Create a display string with title and progress
		var displayInfo []string
		displayInfo = append(displayInfo, entry.Title)

		if entry.Episodes > 0 {
			displayInfo = append(displayInfo, fmt.Sprintf("%d/%d", int(entry.Progress), entry.Episodes))
		} else {
			displayInfo = append(displayInfo, fmt.Sprintf("Progress: %d", int(entry.Progress)))
		}

		if entry.Score > 0 {
			displayInfo = append(displayInfo, fmt.Sprintf("Score: %.1f", entry.Score))
		}

		items[i] = Pair{
			Label: strings.Join(displayInfo, " - "),
			Value: entry.ID,
		}
	}

	selectedID, err := ShowCLIMenu(List, items)
	if err != nil {
		return "", fmt.Errorf("menu error: %w", err)
	}

	return selectedID, nil
}

// ShowMainMenu displays the main menu for Anilist operations
func ShowMainMenu() (string, error) {
	items := []Pair{
		{Label: "Search Anime", Value: "search"},
		{Label: "View My List", Value: "list"},
		{Label: "Sync from Anilist", Value: "sync_from"},
		{Label: "Sync to Anilist", Value: "sync_to"},
		{Label: "Exit", Value: "exit"},
	}

	action, err := ShowCLIMenu(List, items)
	if err != nil {
		return "", fmt.Errorf("menu error: %w", err)
	}

	return action, nil
}

// ShowAnimeUpdateMenu displays a menu for updating anime status/progress
func ShowAnimeUpdateMenu(anime *tracker.AnimeInfo) (string, error) {
	items := []Pair{
		{Label: "Update Status", Value: "status"},
		{Label: "Update Progress", Value: "progress"},
		{Label: "Update Score", Value: "score"},
		{Label: "Back", Value: "back"},
	}

	action, err := ShowCLIMenu(List, items)
	if err != nil {
		return "", fmt.Errorf("menu error: %w", err)
	}

	return action, nil
}

// ShowEpisodeSelection displays a menu to select episode number
func ShowEpisodeSelection(totalEpisodes int) (float64, error) {
	if totalEpisodes <= 0 {
		totalEpisodes = 100 // Default max if total episodes unknown
	}

	items := make([]Pair, totalEpisodes+1)
	for i := 0; i <= totalEpisodes; i++ {
		items[i] = Pair{
			Label: fmt.Sprintf("Episode %d", i),
			Value: fmt.Sprintf("%d", i),
		}
	}

	episodeStr, err := ShowCLIMenu(List, items)
	if err != nil {
		return 0, fmt.Errorf("menu error: %w", err)
	}

	var episode float64
	_, err = fmt.Sscanf(episodeStr, "%f", &episode)
	if err != nil {
		return 0, fmt.Errorf("invalid episode: %w", err)
	}

	return episode, nil
}
