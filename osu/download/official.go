package download

import (
	"context"
	"errors"
	"fmt"
	"github.com/adrg/xdg"
	"github.com/rs/zerolog/log"
	"golang.org/x/oauth2"
	"io"
	"net/http"
	"os"
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
	defaultPath, err := xdg.DataFile("osu")
	if err != nil {
		return "", "", err
	}
	storage, err := os.ReadFile(defaultPath + "/storage.ini")
	ok := err == nil
	if ok {
		lines := strings.Split(string(storage), "\n")
		for _, file := range lines {
			if strings.Contains(file, "FullPath") {
				defaultPath = strings.Replace(strings.Split(file, "=")[1], " ", "", -1)
			}
		}
	}

	config, err := os.ReadFile(defaultPath + "/game.ini")
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

func (d *OfficialDownloader) download(beatmapId int, novideo string) ([]byte, error) {
	// https://github.com/ppy/osu/blob/master/osu.Game/Online/API/Requests/DownloadBeatmapSetRequest.cs#L28
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/beatmapsets/%d/download?noVideo=%s", apiBase, beatmapId, novideo), nil)
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

func (d *OfficialDownloader) DownloadBeatmapset(beatmapsetId int) ([]byte, error) {
	return d.download(beatmapsetId, "")
}

func (d *OfficialDownloader) DownloadBeatmapsetNoVideo(beatmapsetId int) ([]byte, error) {
	return d.download(beatmapsetId, "1")
}

func (d *OfficialDownloader) DownloadBeatmapsetMini(beatmapsetId int) ([]byte, error) {
	return d.download(beatmapsetId, "1")
}
