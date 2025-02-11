# OsuBeatmapSync

A tool to sync osu! beatmaps to OneDrive.

## Build

```shell
$ go build -o osu-beatmap-sync main.go
```

## Usage

- `osu-beatmap-sync`
  - `help` Show help
  - `config` Generate a default config file
  - `login` Login and save tokens to config file
    - `onedrive` Login to OneDrive with specified `client_id`, `client_secret` and `tenant_id` in config file
    - `osu`
      - `local` Save access token and refresh token to config file from local _osu!lazer_ installation
      - `pwd` Login to osu! with username and password
  - `sync` Sync beatmaps to OneDrive
  - `metadata`
    - `make` Generate metadata file for all beatmaps on OneDrive
    - `merge` Merge local metadata (for multi-work)
  - `tool`
    - `xor-hash` Calculate quickXorHash of a file
    - `process` Process beatmap files to No Video & Mini (No Video + No Storyboard) 

## Multi-work

For `metadata make` and `sync`, to work on multiple servers (boosting the speed), you can provide `--tasks <count of tasks>` to split the work into multiple tasks.

For `metadata make`, the task file is `needMakeList<num>.json`, and for `sync`, the task file is `needSync<num>.json`.

You can copy these to another server and run `metadata make` or `sync` with `--worker <num> --start` to work on the same task.

Finally, you need to copy all `metadata.json` to one server and run `metadata merge file1 file2 ...` to merge them.

# Config

```toml
[OneDrive]
client_id = '<your-microsoft-app-client-id>'
client_secret = '<your-microsoft-app-client-secret>'
tenant_id = '<your-microsoft-tenant-id>'

[OneDrive.Token]
# DO NOT fill these fields manually
# use `osu-beatmap-sync login onedrive` to fill
access_token = ''
refresh_token = ''
expires_at = 0

[Osu]
v1_api_key = '<your-osu-legacy-api-key>'
enable_sayobot = true  # https://osu.sayobot.cn/
enable_nerinyan = true  # https://nerinyan.moe/
enable_catboy = true  # https://catboy.best/
enable_official = true  # https://osu.ppy.sh/
process_types = []  # no_video, no_storyboard, no_bg, no_hit_sound, mini

[Osu.Sayobot]
# 自动 -> auto
# 中国电信 -> Telecom
# 中国移动 -> cmcc
# 中国联通 -> unicom
# 腾讯云CDN -> CDN
# 德国 -> DE
# 美国 -> USA
server = 'auto'

[Osu.OfficialDownloader]
# DO NOT fill these fields manually
# use `osu-beatmap-sync login osu local/pwd` to fill
access_token = ''
refresh_token = ''

[Path]
# Level 1
root = 'path/to/your/root'

# Level 2 <mode>
std = 'std'
taiko = 'taiko'
catch = 'catch'
mania = 'mania'

# Level 3 <status>
ranked = 'ranked'
loved = 'loved'
qualified = 'qualified'

[General]
max_concurrent = 36
upload_multiple = 2
log_level = 1  # https://pkg.go.dev/github.com/rs/zerolog#Level
```

## License

MIT License (c) 2025 MingxuanGame
