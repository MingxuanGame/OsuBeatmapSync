package model

type GeneralConfig struct {
	MaxConcurrent  int  `toml:"max_concurrent"`
	UploadMultiple int  `toml:"upload_multiple"`
	LogLevel       int8 `toml:"log_level"`
}

type Config struct {
	OneDrive OneDrive
	Osu      Osu
	Path     OneDrivePath
	General  GeneralConfig
}

type OneDrive struct {
	ClientId     string `toml:"client_id"`
	ClientSecret string `toml:"client_secret"`
	Tenant       string `toml:"tenant_id"`

	Token *Token
}

type Token struct {
	AccessToken  string `toml:"access_token"`
	RefreshToken string `toml:"refresh_token"`
	ExpiresAt    int64  `toml:"expires_at"`
}

type OneDrivePath struct {
	// Level 1
	Root string `toml:"root"`

	// Level 2
	StdPath   string `toml:"std"`
	TaikoPath string `toml:"taiko"`
	CatchPath string `toml:"catch"`
	ManiaPath string `toml:"mania"`

	// Level 3
	RankedPath    string `toml:"ranked"`
	LovedPath     string `toml:"loved"`
	QualifiedPath string `toml:"qualified"`
}

type Osu struct {
	V1ApiKey string `toml:"v1_api_key"`
	Sayobot  struct {
		Server string `toml:"server"`
	}
	OfficialDownloader struct {
		AccessToken  string `toml:"access_token"`
		RefreshToken string `toml:"refresh_token"`
	}
	EnableSayobot  bool     `toml:"enable_sayobot"`
	EnableNerinyan bool     `toml:"enable_nerinyan"`
	EnableCatboy   bool     `toml:"enable_catboy"`
	EnableOfficial bool     `toml:"enable_official"`
	ProcessTypes   []string `toml:"process_types"`
}
