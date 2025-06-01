# To be planned
- Headers in episode video data
- Download system
- Api limit for Anilist and MAL
- Database Migration with new fields (Schema)

# Stack

Logging solution - Zap
Configuration solution - Viper
Flags - Cobra
# Features

Anime specific Tracker
Anime specific Language
Anime specific Source
Subtitles - opensubtitles.org
# UI
- Continue watching (start last immediately)
- Currently watching -> left-right: switch anime, up-down: features
	- Watch anime
	- Update Progress
	- Change Score
	- Change Status
	- Change Source
	- Change Language
	- Change Tracker
- Show All Anime -> Same as Currently watching
- Extensions -> Install new extensions, Install new Repo, Update extensions, Remove extensions, Reinstall extensions
- Settings -> Change Config, Update Rofi theme, Update Pair
- Quit

# Different Aspects
## Image preview division
1. Rofi with image preview
2. Rofi without image preview
3. CLI without image preview.
## Next episode prompt division
1. Rofi with next episode prompt
2. Rofi without next episode prompt
3. CLI with next episode prompt
4. CLI without next episode prompt
## Database storage division
1. Local anime database
2. Upstream anime database
## Upstream support division
- MAL Support
- Anilist Support

# Rofi Working

- Default Themes installed at `.local/share/pair/rofi/`
- User themes at `.config/pair/rofi/`
- Use user themes first if not working then use fallback default themes
## Updating

Config version saved in database, if rofi config version of new pair version > database rofi config version, download new configs, install them.

# Database

## Contents
1. Tracker
2. User Progress
3. Config: Rofi version, Last update, etc
4. Extension path, version, etc
## Working
1. Database takes precedence
2. anime data in `.config/pair/anime_tracking.toml`
```
[[anime]]
title = "Jujutsu Kaisen"
tracking = "local"
current_episode = 5
total_episodes = 24
```
3. if tracker changed, search for anime in the tracker, ask user to select correct anime, if unable to select tracker, give option to search for custom name or use old tracker.
4. `tracking = "tracker:id"` format to specify id of anime, remove it after updating tracker.
5. Do not change db until it is complete.
# Tracker Sync

- User can edit `.config/pair/anime_tracking.toml` for a friendly way to edit.
- Upstream data would take precedence.
- If user edit > database: update
- if user edit < database: do not update
- if upstream > database: update
- if upstream < database: update

# Extensions

## Interface:

### Aniyomi

#### Type def

- Anime datatype
	url -> string
	title -> string
	artist -> string?
	author -> string?
	description -> string?
	genre -> string?
	thumbnail_url -> string?
	Status: -> enum: Unknown, Ongoing, completed, licensed, publishing_finished, cancelled, on_haitus 

-  Episode datatype
	url -> string
	name -> string
	date_upload -> long
	episode_number -> float
	scanlator -> string?

-  Video datatype
	track:
		url, lang
	url -> string
	quality -> string
	videourl -> string?
	headers -> ??
	subtitleTrack -> track
	audioTracks -> track

- AnimePage
	animes -> List < Anime >
	hasNextPage -> Boool

#### Source:
id -> long
name -> string

getAnimeDetails(anime) -> Anime
getEpisodeList(anime) -> List < Episode >
getVideoList(episode) -> List < Video >

#### Catalog
lang -> string
supportsLatest -> bool
supportsRelatedAnime -> Bool

getPopularAnime(page) -> AnimePage
getSearchAnime(page, query, filters) -> AnimePage
getLatestUpdates(page) -> AnimePage
getRelatedAnimeList(anime) -> AnimePage
getFilterList() -> AnimeFilterList

### Pair

Rate limit (per minute)
API Data from scraper (if official API, api data would be structured)

Anime {
	id
	title
	alternative title (list)
	Genre
	Episodes
	Sub/Dub
	description
	thumbnail url
	Tags
	Status
	Author
	Release Year
}

Episode {
	Synopsis
	Title
	Release date
	Stream url
}

**Assuming the following numerical Source IDs:**

- `1001`: Nyaa (Torrent)
- `1002`: Crunchyroll
- `1003`: Funimation

`anime-tool` is binary of an extension.

expected JSON outputs for the `anime-tool`, covering the Aniyomi interface functions and the provided command-line options.

```
anime-tool -h
```

```
Usage: anime-tool [OPTIONS] COMMAND [ARGS]...

A command-line tool for interacting with anime video sources.

Options:
  -h, --help  Show this message and exit.

Commands:
  details       Get detailed information about an anime.
  episodes      Get the list of episodes for an anime.
  extension-info Get information about a specific extension.
  filters       Get the list of available filters for a source.
  latest        Get the latest anime updates from a source.
  list-sources  List all available anime video sources.
  magnet-link   Get the magnet link for a torrent anime episode.
  popular       Get popular anime from a source.
  related       Get related anime for a given anime.
  search        Search for anime on a source.
  source-info   Get information about a specific anime video source.
  stream-url    Get the direct video stream URL for an anime episode.

Examples:

  # Get information about the Crunchyroll extension
  anime-tool extension-info

  # Get information about source with ID 1002 (Crunchyroll)
  anime-tool source-info 1002

  # Get popular anime from source 1002 (Crunchyroll), page 1
  anime-tool popular 1002 --page 1

  # Get latest updates from source 1001 (Nyaa), page 1
  anime-tool latest 1001 --page 1

  # Search for "Attack on Titan" on source 1002 (Crunchyroll), page 1
  anime-tool search 1002 --query "Attack on Titan" --page 1

  # Search with filters on source 1002 (Crunchyroll)
  anime-tool search 1002 --query "" --filters '{"Genres": ["Action"], "Status": "Ongoing"}' --page 1

  # Get details for the anime at the given URL from source 1003 (Funimation)
  anime-tool details 1003 --anime https://www.funimation.com/shows/my-hero-academia/

  # Get episodes for the anime at the given URL from source 1002 (Crunchyroll)
  anime-tool episodes 1002 --anime https://www.crunchyroll.com/frieren-beyond-journeys-end

  # Get stream URL for episode at the given URL from source 1002 (Crunchyroll), episode number (if applicable)
  anime-tool stream-url 1002 --anime https://www.crunchyroll.com/watch/... --episode 1

  # Get magnet link for episode at the given URL from source 1001 (Nyaa)
  anime-tool magnet-link 1001 --anime https://nyaa.si/view/1234567 --episode 1

  # Get available filters for source with ID (replace 'source_id' with actual ID)
  anime-tool filters source_id

  # Get related anime for the anime at the given URL from source 1002 (Crunchyroll), page 1
  anime-tool related 1002 --anime https://www.crunchyroll.com/frieren-beyond-journeys-end --page 1
```

1. Extension info
```
anime-tool extension-info
```

```
  {
    "name": "Crunchyroll",
    "pkg": "anime-crunchyroll",
    "lang": "en",
    "version": "2.1",
    "nsfw": false,
    "sources": [
      {
        "name": "Crunchyroll",
        "lang": "en",
        "id": "a1b2c3d4e5f6",
        "baseUrl": "https://www.crunchyroll.com/"
      }
    ]
  }
```

2. Source info

```
anime-tool source-info 1002
```

```
{
  "id": 1002,
  "name": "Crunchyroll",
  "baseUrl": "https://www.crunchyroll.com/",
  "language": "en",
  "nsfw": false,
  "ratelimit": 20,
  "supportsLatest": true,
  "supportsSearch": true,
  "supportsRelatedAnime": true
}
```

3. Popular anime

```
anime-tool popular 1002 --page 1
```

```
[
  {
    "anime_id": "https://www.crunchyroll.com/frieren-beyond-journeys-end",
    "title": "Frieren: Beyond Journey's End",
    "artist": null,
    "author": null,
    "description": "After the hero defeated the Demon King, the long-lived elf Frieren lived on...",
    "genre": "Adventure, Drama, Fantasy",
    "thumbnail_url": "https://www.crunchyroll.com/imgsrv/...",
    "status": "Ongoing"
  },
  {
    "anime_id": "https://www.crunchyroll.com/one-piece",
    "title": "One Piece",
    "artist": "Eiichiro Oda",
    "author": "Eiichiro Oda",
    "description": "Monkey D. Luffy refuses to let anyone or anything stand in the way of his quest to become the king of all pirates...",
    "genre": "Action, Adventure, Comedy, Fantasy, Shounen",
    "thumbnail_url": "https://www.crunchyroll.com/imgsrv/...",
    "status": "Ongoing"
  }
  // ... more popular anime
]
```

4. Get Latest updates

```
anime-tool latest 1001 --page 1
```

```
[
  {
    "anime_id": "https://nyaa.si/view/7890123",
    "title": "[MTBB] Vinland Saga S2 - 24 (1080p) [YZA]",
    "artist": null,
    "author": "Makoto Yukimura",
    "description": "The final episode of Vinland Saga Season 2.",
    "genre": "Action, Adventure, Drama, Historical, Seinen",
    "thumbnail_url": null, // Torrent sources might not have thumbnails
    "status": "Completed"
  },
  {
    "anime_id": "https://nyaa.si/view/4567890",
    "title": "[LoliHouse] To Your Eternity S2 - 20 (1080p) [BCD]",
    "artist": null,
    "author": "Yoshitoki Ōima",
    "description": "The twentieth episode of To Your Eternity Season 2.",
    "genre": "Adventure, Drama, Fantasy, Shounen, Supernatural",
    "thumbnail_url": null,
    "status": "Ongoing"
  }
  // ... more latest torrents
]
```

5. Search Anime

```
anime-tool search 1002 --query "Attack on Titan" --page 1
```

```
[
  {
    "animd_id": "https://www.crunchyroll.com/attack-on-titan",
    "title": "Attack on Titan",
    "artist": "Hajime Isayama",
    "author": "Hajime Isayama",
    "description": "Humanity has been devastated by the titans. Eren Yeager vows to eliminate all of them...",
    "genre": "Action, Drama, Fantasy, Horror, Shounen",
    "thumbnail_url": "https://www.crunchyroll.com/imgsrv/...",
    "status": "Completed"
  },
  {
    "anime_id": "https://www.crunchyroll.com/attack-on-titan-final-season",
    "title": "Attack on Titan Final Season Part 1",
    "artist": "Hajime Isayama",
    "author": "Hajime Isayama",
    "description": "The fourth and final season of Attack on Titan.",
    "genre": "Action, Drama, Fantasy, Horror, Military, Shounen",
    "thumbnail_url": "https://www.crunchyroll.com/imgsrv/...",
    "status": "Completed"
  }
  // ... more search results
]
```

```
anime-tool search 1002 --query "" --filters '{"Genres": ["Action"], "Status": "Ongoing"}' --page 1
```

```
[
  {
    "anime_id": "https://www.crunchyroll.com/some-ongoing-action-anime-1",
    "title": "Ongoing Action Anime Title 1",
    "artist": "...",
    "author": "...",
    "description": "...",
    "genre": "Action, Fantasy",
    "thumbnail_url": "...",
    "status": "Ongoing"
  },
  {
    "anime_id": "https://www.crunchyroll.com/another-ongoing-action-anime-2",
    "title": "Another Ongoing Action Anime",
    "artist": "...",
    "author": "...",
    "description": "...",
    "genre": "Action, Sci-Fi",
    "thumbnail_url": "...",
    "status": "Ongoing"
  }
  // ... more anime matching the filters
]
```

6. Get anime details

```
anime-tool details 1003 --anime anime_id
```

```
{
  "anime_id": "https://www.funimation.com/shows/my-hero-academia/",
  "title": "My Hero Academia",
  "artist": "Kohei Horikoshi",
  "author": "Kohei Horikoshi",
  "description": "Izuku Midoriya wants to be a hero more than anything...",
  "genre": "Action, Adventure, Comedy, Sci-Fi, Shounen",
  "thumbnail_url": "https://www.funimation.com/...",
  "status": "Ongoing"
}
```

7. Get Episode list

```
anime-tool episodes 1002 --anime anime_id
```

```
[
  {
    "anime_id": "https://www.crunchyroll.com/watch/...",
    "name": "The End of the Journey",
    "date_upload": 1695980400, // Example timestamp for 2023-09-29T15:00:00Z
    "episode_number": 1.0,
    "scanlator": null
  },
  {
    "anime_id": "https://www.crunchyroll.com/watch/...",
    "name": "It Didn't Feel Real at All",
    "date_upload": 1701942000, // Example timestamp for 2023-10-06T15:00:00Z
    "episode_number": 2.0,
    "scanlator": null
  }
  // ... more episodes
]
```

8. Get Video List (Stream link)

```
anime-tool stream-url 1002 --anime anime_id --episode int
```

```
{
  "streams": [
    {
      "anime_id": "https://www.crunchyroll.com/watch/...",
      "quality": "1080p",
      "videourl": "https://stream.crunchyroll.com/...",
      "headers": {},
      "subtitleTrack": {
        "url": "https://static.crunchyroll.com/...",
        "lang": "en"
      },
      "audioTracks": [
        {
          "url": null,
          "lang": "ja"
        },
        {
          "url": null,
          "lang": "en"
        }
      ]
    },
    {
      "anime_id": "https://www.crunchyroll.com/watch/...",
      "quality": "720p",
      "videourl": "https://stream.crunchyroll.com/...",
      "headers": {},
      "subtitleTrack": {
        "url": "https://static.crunchyroll.com/...",
        "lang": "en"
      },
      "audioTracks": [
        {
          "url": null,
          "lang": "ja"
        },
        {
          "url": null,
          "lang": "en"
        }
      ]
    }
    // ... more streams
  ],
  "subtitles": [
    {
      "url": "https://static.crunchyroll.com/...",
      "lang": "en"
    },
    {
      "url": "https://static.crunchyroll.com/...",
      "lang": "es"
    },
    {
      "url": "https://static.crunchyroll.com/...",
      "lang": "fr"
    }
  ]
}
```

Torrent Source (e.g., Nyaa):

```
anime-tool magnet-link 1001 --anime anime_id --episode int
```

```
{
  "magnetLink": "magnet:?xt=urn:btih:abcdef1234567890abcdef1234567890abcdef12&dn=[SubsPlease]%20Frieren%20-%2027%20(1080p)%20[ABC]&tr=...&tr=..."
}
```

9.  Get Filter List

```
anime-tool filters source_id
```

```
{
  "filters": [
    {
      "type": "header",
      "text": "Filter results"
    },
    {
      "type": "group",
      "name": "Genres",
      "entries": [
        {"name": "Action", "state": false},
        {"name": "Adventure", "state": false},
        {"name": "Comedy", "state": false},
        {"name": "Drama", "state": false},
        {"name": "Fantasy", "state": false},
        // ... more anime genres
        {"name": "Slice of Life", "state": false}
      ]
    },
    {
      "type": "select",
      "name": "Season",
      "options": ["All", "Spring 2024", "Winter 2024", "Fall 2023", "Summer 2023"],
      "selectedValue": "All"
    },
    {
      "type": "select",
      "name": "Sort",
      "options": ["Popularity", "Newest", "Alphabetical"],
      "selectedValue": "Popularity"
    },
    {
      "type": "checkbox",
      "name": "Subtitles",
      "state": false
    },
    {
      "type": "checkbox",
      "name": "Dubbed",
      "state": false
    }
    // ... other anime-specific filters
  ]
}
```

10. Get Related Anime

```
anime-tool related 1002 --anime anime_id --page 1
```

```
[
  {
    "anime_id": "https://www.crunchyroll.com/sousou-no-frieren",
    "title": "Frieren: Beyond Journey's End (Same Series)",
    "artist": null,
    "author": null,
    "description": "The official page for Frieren: Beyond Journey's End.",
    "genre": "Adventure, Drama, Fantasy",
    "thumbnail_url": "https://www.crunchyroll.com/imgsrv/...",
    "status": "Ongoing"
  },
  {
    "anime_id": "https://www.crunchyroll.com/another-related-anime",
    "title": "Another Related Anime",
    "artist": null,
    "author": null,
    "description": "Description of a related anime.",
    "genre": "Action, Fantasy",
    "thumbnail_url": "https://www.crunchyroll.com/imgsrv/...",
    "status": "Completed"
  }
  // ... more related anime
]
```

11. Error handling

```
{
  "error": {
    "message": "An unexpected error occurred.",
    "code": 500
  }
}
```

## Installation

- Installed through Extension setting in main menu, or config file.
- Installation link would be a github/gitlab link to a json file containing various source provided like:
```
[{
	name: {name of extension}
	pkg: {binary name}
	lang: {language support}
	version: {binary version}
	nsfw: {bool}
	sources: [{
		name: {website name}
		lang: {language support}
		id: {(name + lang + baseUrl).hash()}
		baseUrl: {base url of website} 
	}]
},
{}
]
```
- link to json for example: `https://raw.githubusercontent.com/{username}/{repo}/{branch}/{filename}.json`
- binaries would be at `https://github.com/{username}/{repo}/blob/{branch}/bin/{pkg}`
- binaries would be installed at `$HOME/.local/share/pair/extensions/{pkg}`

## Updating
- binaries would be updated when user runs extension update in settings
- Version information of binaries would be stored in the database, it would be checked with binary name (binary name contains the version information), if any discrepancy is found, update extension and update database.

# Discord RPC

Same as curd, image of anime
Name of anime, episode - {time} [if paused: (paused)]
Buttons: View on anilist, View on MAL