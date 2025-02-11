package main

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"github.com/MingxuanGame/OsuBeatmapSync/application"
	"github.com/MingxuanGame/OsuBeatmapSync/base_service"
	cli2 "github.com/MingxuanGame/OsuBeatmapSync/cli"
	"github.com/MingxuanGame/OsuBeatmapSync/onedrive/quickxorhash"
	"github.com/MingxuanGame/OsuBeatmapSync/utils/beatmap_processing"
	"github.com/urfave/cli/v3"
	"os"
	"strings"
	"time"
)

func main() {
	base_service.CreateLog()
	defer func(logFile *os.File) {
		err := logFile.Close()
		if err != nil {
			fmt.Println("Failed to close log file:", err)
		}
	}(base_service.LogFile)
	ctx := application.CreateSignalCancelContext()

	cmd := &cli.Command{
		Name:  "osu-beatmap-sync",
		Usage: "Sync osu! beatmaps to OneDrive",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return nil
		},
		EnableShellCompletion: true,
		Commands: []*cli.Command{
			{
				Name:  "login",
				Usage: "Login to OneDrive",
				Commands: []*cli.Command{
					{
						Name:  "onedrive",
						Usage: "Login to OneDrive",
						Action: func(ctx context.Context, cmd *cli.Command) error {
							config, err := base_service.LoadConfig()
							if err != nil {
								return err
							}
							_, err = application.Login(&config, ctx)
							return err
						},
					},
					{
						Name:  "osu",
						Usage: "Login to osu!",
						Commands: []*cli.Command{
							{
								Name:  "local",
								Usage: "Login to osu! using local osu!lazer",
								Action: func(context.Context, *cli.Command) error {
									return cli2.LoginToOsuUseLocal()
								},
							},
							{
								Name:  "pwd",
								Usage: "Login to osu! using username and password",
								Action: func(ctx context.Context, cmd *cli.Command) error {
									var username string
									var password string
									fmt.Printf("Username: ")
									_, err := fmt.Scanln(&username)
									if err != nil {
										return err
									}
									fmt.Printf("Password: ")
									_, err = fmt.Scanln(&password)
									if err != nil {
										return err
									}
									if username == "" || password == "" {
										return fmt.Errorf("username or password not specified")
									}
									return cli2.LoginToOsu(ctx, username, password)
								},
							},
						},
					},
				},
			},
			{
				Name:  "config",
				Usage: "Generate config file",
				Action: func(context.Context, *cli.Command) error {
					err := cli2.GenerateConfig()
					if err != nil {
						return err
					}
					fmt.Println("Config file generated successfully")
					return nil
				},
			},
			{
				Name:  "metadata",
				Usage: "query & manage metadata",
				Commands: []*cli.Command{
					{
						Name:  "make",
						Usage: "make all metadata",
						Flags: []cli.Flag{
							&cli.BoolFlag{Name: "start", Aliases: []string{"s"}, Value: false, Usage: "when execute sub-task, start worker"},
							&cli.IntFlag{Name: "tasks", Aliases: []string{"t"}, Value: 1, Usage: "split tasks into n files"},
							&cli.IntFlag{Name: "worker", Aliases: []string{"w"}, Value: 0, Usage: "execute sub-task from n file"},
						},
						Action: func(ctx context.Context, cmd *cli.Command) error {
							return cli2.MakeMetadata(ctx, int(cmd.Int("tasks")), int(cmd.Int("worker")), cmd.Bool("start"))
						},
					},
					{
						Name:  "merge",
						Usage: "merge metadata files",
						Flags: []cli.Flag{
							&cli.BoolFlag{Name: "upload", Aliases: []string{"u"}, Value: false, Usage: "upload merged metadata to OneDrive"},
						},
						Action: func(ctx context.Context, cmd *cli.Command) error {
							return cli2.MergeMetadata(ctx, cmd.Bool("upload"), cmd.Args().Slice())
						},
					},
				},
			}, {
				Name:  "sync",
				Usage: "sync all beatmaps",
				Flags: []cli.Flag{
					//&cli.BoolFlag{Name: "force", Aliases: []string{"f"}, Value: false},
					&cli.BoolFlag{Name: "start", Aliases: []string{"s"}, Value: false, Usage: "when execute sub-task, start worker"},
					&cli.IntFlag{Name: "tasks", Aliases: []string{"t"}, Value: 1, Usage: "split tasks into n files"},
					&cli.IntFlag{Name: "worker", Aliases: []string{"w"}, Value: 0, Usage: "execute sub-task from n file"},
					&cli.TimestampFlag{Name: "since", Config: cli.TimestampConfig{
						Timezone: time.Local,
						Layouts:  []string{time.DateTime, time.DateOnly, time.RFC3339},
					}, Usage: "sync beatmaps since the specified time", Value: time.Now()},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					err := cli2.SyncBeatmaps(ctx, int(cmd.Int("tasks")), int(cmd.Int("worker")), cmd.Bool("start"), cmd.Timestamp("since"))
					if err != nil {
						return err
					}
					return nil
				},
			},
			{
				Name:  "tool",
				Usage: "tool for beatmap",
				Commands: []*cli.Command{
					{
						Name:  "process",
						Usage: "process beatmap to no-video and mini",
						Flags: []cli.Flag{
							&cli.BoolFlag{Name: "no-video", Aliases: []string{"nv"}, Value: false, Usage: "remove video"},
							&cli.BoolFlag{Name: "mini", Aliases: []string{"m"}, Value: false, Usage: "mini"},
							&cli.BoolFlag{Name: "no-hitsound", Aliases: []string{"nh"}, Value: false, Usage: "remove hit sound"},
							&cli.BoolFlag{Name: "no-storyboard", Aliases: []string{"ns"}, Value: false, Usage: "remove storyboard"},
							&cli.BoolFlag{Name: "no-bg", Aliases: []string{"nb"}, Value: false, Usage: "remove background image"},
						},
						Action: func(ctx context.Context, cmd *cli.Command) error {
							files := cmd.Args().Slice()
							if len(files) == 0 {
								return fmt.Errorf("no file specified")
							}
							for _, arg := range files {
								data, err := os.ReadFile(arg)
								if err != nil {
									fmt.Println(err)
									continue
								}
								var process []beatmap_processing.Processor
								if cmd.Bool("no-video") {
									process = append(process, beatmap_processing.NewNoVideoProcessor())
								}
								if cmd.Bool("mini") {
									process = append(process, beatmap_processing.NewMiniProcessor())
								}
								if cmd.Bool("no-hitsound") {
									process = append(process, beatmap_processing.NewNoHitSoundProcessor())
								}
								if cmd.Bool("no-storyboard") {
									process = append(process, beatmap_processing.NewNoStoryboardProcessor())
								}
								if cmd.Bool("no-bg") {
									process = append(process, beatmap_processing.NewNoBackgroundProcessor())
								}
								for _, typ := range process {
									processed, err := beatmap_processing.Process(typ, data)
									filename := fmt.Sprintf("%s [%s].osz", strings.TrimSuffix(arg, ".osz"), typ)
									err = os.WriteFile(filename, processed, 0666)
									if err != nil {
										fmt.Println(err)
										continue
									}
									fmt.Printf("Processed: %s\n", filename)
								}
							}
							return nil
						},
					},
					{
						Name:  "xor-hash",
						Usage: "calculate xor hash of file",
						Flags: []cli.Flag{
							&cli.StringFlag{Name: "hash", Aliases: []string{"s"}, Usage: "hash to compare"},
							&cli.BoolFlag{Name: "base64", Aliases: []string{"b"}, Usage: "hash is base64 encoded"},
						},
						Action: func(ctx context.Context, cmd *cli.Command) error {
							filename := cmd.Args().First()
							if filename == "" {
								return fmt.Errorf("no file specified")
							}
							data, err := os.ReadFile(filename)
							if err != nil {
								return err
							}
							targetHash := quickxorhash.Sum(data)
							sourceHash := cmd.String("hash")
							if sourceHash == "" {
								fmt.Println(targetHash)
								return nil
							}
							if cmd.Bool("base64") {
								decoded, err := base64.StdEncoding.DecodeString(sourceHash)
								if err != nil {
									return err
								}
								sourceHash = hex.EncodeToString(decoded)
							}
							if sourceHash == targetHash {
								fmt.Println("Match")
							} else {
								fmt.Println("Not Match")
							}
							fmt.Printf("Source: %s\n", sourceHash)
							fmt.Printf("Target: %s\n", targetHash)
							return nil
						},
					},
				},
			},
		},
	}

	if err := cmd.Run(ctx, os.Args); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

}
