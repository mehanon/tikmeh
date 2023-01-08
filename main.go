package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/mehanon/tikmeh/tikwm"
	"github.com/urfave/cli/v2"
	"log"
	"os"
	"strings"
	"time"
)

const (
	PackageName             = "Tikmeh"
	VersionInfo             = "0.2.3 (Jan 8, 2023)"
	GithubLink              = "https://github.com/mehanon/tikmeh"
	DefaultWorkingDirectory = "."
)

func main() {
	defer func() {
		if err := recover(); err != nil {
			log.Printf("%v", err)
			println("\nPress any button to exit...")
			_, _ = bufio.NewReader(os.Stdin).ReadByte()
		}
	}()

	if len(os.Args) > 1 { // with console args
		Cli := NewTikmehCli()
		if err := Cli.Run(os.Args); err != nil {
			log.Fatal(err)
		}
	} else { // interactive mode
		fmt.Printf("%s (%s) [sources and up-to-date executables: %s]\n"+
			"Enter 'help' to get help message.\n", PackageName, VersionInfo, GithubLink)
		reader := bufio.NewReader(os.Stdin)
		for {
			print(">>> ")
			input, err := reader.ReadString('\n')
			if err != nil {
				log.Fatalf(err.Error())
			}
			input = strings.Trim(input, " \n\t")
			if input == "" {
				println("see you next time (exiting in 5 sec)")
				time.Sleep(time.Second * 5)
				os.Exit(0)
			}
			Cli := NewTikmehCli()
			if err := Cli.Run(append([]string{PackageName}, strings.Split(input, " ")...)); err != nil {
				log.Fatal(err)
			}
		}
	}
}

type jsonOrOutput struct {
	Files []string `json:"files"`
	Error string   `json:"error"`
}

type output struct {
	IsJson bool
	Json   jsonOrOutput
}

func (out *output) file(file, message string) {
	if out.IsJson {
		out.Json.Files = append(out.Json.Files, file)
	} else {
		log.Println(message)
	}

}
func (out *output) err(err, message string) {
	if out.IsJson {
		out.Json.Error = err
	} else {
		log.Println(message)
	}
}

func NewTikmehCli() *cli.App {
	return &cli.App{
		Name:    PackageName,
		Usage:   "TikTok downloader",
		Version: fmt.Sprintf("%s [source code: %s]", VersionInfo, GithubLink),
		Description: "Download TikTok videos in the best quality.\n" +
			fmt.Sprintf("Examples:\n"+
				"  %s                                                     start in interactive mode\n"+
				"  %s tiktok.com/@shrimpydimpy/video/7133412834960018730  simply download a tiktok to the current directory\n"+
				"  %s --convert --directory goddess 7133412834960018730   download to ./goddess and convert to h.264\n"+
				"  %s --convert profile shrimpydimpy                      download shrimpydimpy videos and convert them to h.264\n"+
				"(more help on downloading profiles -- %s profile help)",
				os.Args[0], os.Args[0], os.Args[0], os.Args[0], os.Args[0]),
		Action: func(ctx *cli.Context) error {
			out := &output{IsJson: ctx.Bool("json"), Json: jsonOrOutput{Files: []string{}}}
			defer func() {
				if out.IsJson {
					buffer, err := json.MarshalIndent(out.Json, "", "    ")
					if err != nil {
						log.Fatalln(err.Error())
					}
					println(string(buffer))
				}
			}()

			if ctx.String("directory") == "" {
				err := ctx.Set("directory", DefaultWorkingDirectory)
				if err != nil {
					out.err(err.Error(), err.Error())
					return nil
				}
			}
			if _, err := os.Stat(ctx.String("directory")); os.IsNotExist(err) {
				err := os.Mkdir(ctx.String("directory"), 0777)
				if err != nil {
					output := fmt.Sprintf("while creating directory %s, an error occured: %s", ctx.String("directory"), err.Error())
					out.err(output, output)
					return nil
				}
			}

			for _, video := range ctx.Args().Slice() {
				filename, err := tikwm.DownloadTiktok(video, ctx.String("directory"))
				if err != nil {
					output := fmt.Sprintf("while downloading %s, an error occured: %s", video, err.Error())
					out.err(output, output)
					return nil
				}
				if ctx.Bool("convert") {
					h264, err := tikwm.ConvertToH264(filename, ctx.String("ffmpeg"))
					if err != nil {
						output := fmt.Sprintf("while converting %s, an error occured: %s", filename, err.Error())
						out.err(output, output)
						return nil
					}
					err = os.Rename(h264, filename)
					if err != nil {
						output := fmt.Sprintf("while converting %s, an error occured: %s", filename, err.Error())
						out.err(output, output)
						return nil
					}
				}
				out.file(filename, fmt.Sprintf("downloaded %s", filename))
			}

			return nil
		},
		Commands: []*cli.Command{{
			Name:    "profile",
			Usage:   fmt.Sprintf("Downloads all videos of a TikTok user ('%s profile help' - more info)", os.Args[0]),
			Aliases: []string{"p"},
			Description: "Download all videos of a TikTok user, until already downloaded is met\n" +
				fmt.Sprintf("Examples:\n"+
					"  %s profile shrimpydimpy losertron   download all their videos to ./shrimpydimpy & ./losertron accordinaly\n"+
					"  %s -d . -c profile --all losertron  download all shrimpydimpy videos to current directory, convert to h.264", os.Args[0], os.Args[0]),
			Flags: []cli.Flag{&cli.BoolFlag{
				Name:     "all",
				Aliases:  []string{"a"},
				Value:    false,
				Usage:    "don't stop when an already downloaded video is met, to ensure everything is downloaded",
				Category: "profile",
			}},
			Action: func(ctx *cli.Context) error {
				out := output{IsJson: ctx.Bool("json"), Json: jsonOrOutput{Files: []string{}}}
				defer func() {
					if out.IsJson {
						buffer, err := json.MarshalIndent(out.Json, "", "    ")
						if err != nil {
							log.Fatalln(err.Error())
						}
						println(string(buffer))
					}
				}()

				for _, username := range ctx.Args().Slice() {
					downloader := tikwm.ProfileDownloader{
						Username:   username,
						Directory:  ctx.String("directory"),
						FfmpegPath: ctx.String("ffmpeg"),
						CheckAll:   ctx.Bool("all"),
						Convert:    ctx.Bool("convert"),
					}
					if downloader.Directory == "" {
						downloader.Directory = strings.ToLower(strings.Trim(username, "@ "))
					}
					for resp := range downloader.DownloadIteratively() {
						switch resp.(type) {
						case string:
							out.file(resp.(string), fmt.Sprintf("downloaded %s", resp.(string)))
						case error:
							out.err(resp.(error).Error(), fmt.Sprintf("while downloading %s, an error occuer %s", username, resp.(error).Error()))
							return nil
						}
					}
				}

				return nil
			},
		}},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "directory",
				Aliases:     []string{"d"},
				Value:       "",
				Usage:       "target directory (created if not found)",
				DefaultText: "<dir>=username of the profile, or current for videos",
			}, &cli.BoolFlag{
				Name:    "convert",
				Aliases: []string{"c"},
				Value:   false,
				Usage:   "convert uploaded files to h264 with ffmpeg",
			}, &cli.BoolFlag{
				Name:    "json",
				Aliases: []string{"j"},
				Value:   false,
				Usage:   "print output as json",
			}, &cli.StringFlag{
				Name:    "ffmpeg",
				Aliases: []string{"f"},
				Value:   "ffmpeg",
				Usage:   "path to ffmpeg, (ffmpeg isn't required, unless you use 'convert')",
			},
		},
	}
}
