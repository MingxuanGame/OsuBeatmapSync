package download

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"github.com/MingxuanGame/OsuBeatmapSync/utils"
	"github.com/rs/zerolog/log"
	"golang.org/x/oauth2"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const apiVersion = "20250118"
const apiBase = "https://osu.ppy.sh/api/v2"

// https://github.com/ppy/osu/blob/master/osu.Game/Online/ProductionEndpointConfiguration.cs#L11-L12
const clientID = "5"
const clientSecret = "FGc9GAtyHzeQDshWP5Ah7dega8hJACAJpQtw6OXk"

var officialLogger = log.With().Str("module", "osu.download.official").Logger()

type OfficialDownloader struct {
	client       *http.Client
	ctx          context.Context
	AccessToken  string
	RefreshToken string
}

func NewOfficialDownloader(ctx context.Context, accessToken, refreshToken string) (*OfficialDownloader, error) {
	cfg := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://osu.ppy.sh/oauth/authorize",
			TokenURL: "https://osu.ppy.sh/oauth/token",
		},
		Scopes: []string{"*"},
	}
	tok, err := cfg.TokenSource(ctx, &oauth2.Token{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}).Token()
	if err != nil {
		return nil, err
	}
	return &OfficialDownloader{
		client:       cfg.Client(ctx, tok),
		ctx:          ctx,
		AccessToken:  tok.AccessToken,
		RefreshToken: tok.RefreshToken,
	}, nil
}

func NewOfficialDownloaderLogin(ctx context.Context, username, password string) (*OfficialDownloader, error) {
	cfg := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://osu.ppy.sh/oauth/authorize",
			TokenURL: "https://osu.ppy.sh/oauth/token",
		},
		Scopes: []string{"*"},
	}
	tok, err := cfg.PasswordCredentialsToken(ctx, username, password)
	if err != nil {
		return nil, err
	}
	return &OfficialDownloader{
		client:       cfg.Client(ctx, tok),
		ctx:          ctx,
		AccessToken:  tok.AccessToken,
		RefreshToken: tok.RefreshToken,
	}, nil
}

// GetAccessTokenFromLocal get the access token & refresh token from local osu!lazer
func GetAccessTokenFromLocal() (string, string, error) {
	defaultPath := utils.XDGDataHome("osu")
	storage, err := os.Open(filepath.Join(defaultPath, "storage.ini"))
	defer func(storage *os.File) {
		err := storage.Close()
		if err != nil {
			officialLogger.Error().Err(err).Msg("failed to close storage file")
		}
	}(storage)
	ok := err == nil
	storageReader := bufio.NewReader(storage)
	if ok {
		for {
			b, _, err := storageReader.ReadLine()
			if err != nil {
				return "", "", err
			}
			file := string(b)
			if strings.Contains(file, "FullPath") {
				defaultPath = strings.Replace(strings.Split(file, "=")[1], " ", "", -1)
				break
			}
		}
	}

	config, err := os.ReadFile(filepath.Join(defaultPath, "game.ini"))
	if err != nil {
		return "", "", err
	}
	lines := strings.Split(string(config), "\n")
	for _, file := range lines {
		if strings.HasPrefix(file, "Token") {
			// https://github.com/ppy/osu/blob/master/osu.Game/Online/API/OAuth.cs#L23-L27
			// https://github.com/ppy/osu/blob/master/osu.Game/Online/API/OAuthToken.cs#L38-L51
			oauthToken := strings.Replace(strings.Split(file, "=")[1], " ", "", -1)
			split := strings.Split(oauthToken, "|")
			if len(split) != 3 {
				return "", "", errors.New("invalid token")
			}
			return split[0], split[2], nil
		}
	}
	return "", "", errors.New("token not found")
}

func (d *OfficialDownloader) Name() string {
	return "official"
}

func (d *OfficialDownloader) DownloadBeatmapset(beatmapId int) ([]byte, error) {
	// https://github.com/ppy/osu/blob/master/osu.Game/Online/API/Requests/DownloadBeatmapSetRequest.cs#L28
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/beatmapsets/%d/download", apiBase, beatmapId), nil)
	if err != nil {
		return nil, err
	}
	officialLogger.Trace().Msgf("Requesting %s %s", req.Method, req.URL.String())
	req.Header.Set("x-api_version", apiVersion)
	resp, err := d.client.Do(req)
	if err != nil {
		return nil, err
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		// some beatmap cannot be downloaded (like https://osu.ppy.sh/beatmapsets/30877, DMCA takedown)
		// other api maybe return server error (5xx)
		// osu!api return empty body
		return nil, fmt.Errorf("empty body")
	}
	return data, nil
}
