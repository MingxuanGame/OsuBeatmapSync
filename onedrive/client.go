package onedrive

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	. "github.com/MingxuanGame/OsuBeatmapSync/model"
	"github.com/MingxuanGame/OsuBeatmapSync/utils"
	"golang.org/x/oauth2"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

const AuthUrl string = "https://login.microsoftonline.com/%s/oauth2/v2.0/authorize"
const TokenUrl string = "https://login.microsoftonline.com/%s/oauth2/v2.0/token"
const RootUrl string = "https://graph.microsoft.com/v1.0"

type GraphClient struct {
	Config *OneDrive
	*http.Client
	ctx context.Context
}

type BatchReq struct {
	Id     string `json:"id"`
	Method string `json:"method"`
	Url    string `json:"url"`
	Dep    string `json:"dependsOn,omitempty"`
}

type BatchResp struct {
	Id      string            `json:"id"`
	Status  int               `json:"status"`
	Headers map[string]string `json:"headers"`
	Body    string            `json:"body"`
}

func NewGraphClient(clientId string, clientSecret string, tenant string, ctx context.Context) (*GraphClient, error) {
	conf := &oauth2.Config{
		ClientID:     clientId,
		ClientSecret: clientSecret,
		Scopes:       []string{"Files.ReadWrite.All", "offline_access"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  fmt.Sprintf(AuthUrl, tenant),
			TokenURL: fmt.Sprintf(TokenUrl, tenant),
		},
		RedirectURL: "http://localhost:8080/callback",
	}
	url := conf.AuthCodeURL("state", oauth2.AccessTypeOffline)
	log.Printf("Please visit here to login: %v\n", url)

	var code string
	called := make(chan struct{})
	mux := http.NewServeMux()
	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		code = r.URL.Query().Get("code")
		_, _ = w.Write([]byte("You can now close this tab."))
		called <- struct{}{}
	})
	go func() {
		err := server.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Println("Failed to start server:", err)
		}
	}()

	<-called
	err := server.Shutdown(ctx)
	if err != nil {
		return nil, err
	}
	tok, err := conf.Exchange(ctx, code)
	if err != nil {
		return nil, err
	}

	client := conf.Client(ctx, tok)
	return &GraphClient{
		&OneDrive{
			ClientId:     clientId,
			ClientSecret: clientSecret,
			Tenant:       tenant,
			Token: &Token{
				AccessToken:  tok.AccessToken,
				RefreshToken: tok.RefreshToken,
				ExpiresAt:    tok.Expiry.Unix(),
			},
		},
		client,
		ctx,
	}, nil
}

func NewExistedGraphClient(onedrive *OneDrive, ctx context.Context) *GraphClient {
	conf := &oauth2.Config{
		ClientID:     onedrive.ClientId,
		ClientSecret: onedrive.ClientSecret,
		Scopes:       []string{"Files.ReadWrite.All"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  fmt.Sprintf(AuthUrl, onedrive.Tenant),
			TokenURL: fmt.Sprintf(TokenUrl, onedrive.Tenant),
		},
		RedirectURL: "http://localhost:8080/callback",
	}
	tok := &oauth2.Token{
		AccessToken:  onedrive.Token.AccessToken,
		RefreshToken: onedrive.Token.RefreshToken,
		Expiry:       time.Unix(onedrive.Token.ExpiresAt, 0),
		TokenType:    "Bearer",
	}
	client := conf.Client(ctx, tok)
	return &GraphClient{
		onedrive,
		client,
		ctx,
	}
}

func (client *GraphClient) NewRequest(method, url string, data []byte) (*http.Request, error) {
	return client.NewRequestWithBuffer(method, url, bytes.NewReader(data))
}

func (client *GraphClient) NewRequestWithBuffer(method, url string, data io.Reader) (*http.Request, error) {
	req, err := http.NewRequestWithContext(client.ctx, method, RootUrl+url, data)
	if err != nil {
		return nil, err
	}
	return req, nil
}

func (client *GraphClient) Do(req *http.Request) (*http.Response, error) {
	resp, err := client.Client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == 429 {
		retryAfter, ctx, err := utils.GetLimitSecond(resp.Header.Get("Retry-After"), client.ctx)
		if err != nil {
			return nil, err
		}
		client.ctx = ctx
		log.Printf("[onedrive] Rate limited, sleeping for %s.\n", retryAfter)
		time.Sleep(retryAfter)
		body, err := io.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}
		retryUrl, ok := strings.CutPrefix(req.URL.String(), RootUrl)
		if !ok {
			retryUrl = req.URL.String()
		}
		req, err := client.NewRequest(req.Method, retryUrl, body)
		return client.Do(req)
	}
	return resp, nil
}

func (client *GraphClient) BatchDo(reqs []BatchReq) ([]BatchResp, error) {
	jsonData, err := json.Marshal(struct {
		Requests []BatchReq `json:"requests"`
	}{
		Requests: reqs,
	})
	if err != nil {
		return nil, err
	}
	req, err := client.NewRequest("POST", "/$batch", jsonData)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("error: %d", resp.StatusCode)
	}
	var response struct {
		Response []BatchResp `json:"responses"`
	}
	return response.Response, nil
}
