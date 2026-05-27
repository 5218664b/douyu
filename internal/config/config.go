package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Room    RoomConfig
	Video   VideoConfig
	Stream  StreamConfig
	Danmaku DanmakuConfig
	Notify  NotifyConfig
	API     APIConfig
}

type RoomConfig struct {
	URL string
}

type VideoConfig struct {
	SourceDir string
	Formats   []string
	URLs      []string
}

type StreamConfig struct {
	RTMPURL    string
	StreamKey  string
	FFmpegPath string
	FFmpegLogLevel string
	CopyVideo  bool
	CopyAudio  bool
	LoopSingleInput bool
}

type DanmakuConfig struct {
	Enabled       bool
	CommandPrefix string
}

type NotifyConfig struct {
	Enabled         bool
	SMTPHost        string
	SMTPPort        string
	Username        string
	Password        string
	From            string
	To              []string
	SubjectPrefix   string
	CooldownSeconds int
}

type APIConfig struct {
	ListenAddr string
}

func Load(path string) (Config, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read %s: %w", path, err)
	}

	cfg, err := parse(content)
	if err != nil {
		return Config{}, err
	}

	applyEnv(&cfg)

	if err := validate(cfg); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func parse(content []byte) (Config, error) {
	cfg := Config{
		Video: VideoConfig{
			Formats: []string{".ts", ".mp4"},
		},
		Stream: StreamConfig{
			FFmpegPath: "/usr/bin/ffmpeg",
			FFmpegLogLevel: "warning",
			CopyVideo:  true,
			CopyAudio:  true,
			LoopSingleInput: false,
		},
		API: APIConfig{
			ListenAddr: "127.0.0.1:8080",
		},
		Danmaku: DanmakuConfig{
			Enabled:       true,
			CommandPrefix: "#",
		},
		Notify: NotifyConfig{
			SubjectPrefix:   "[douyu-streamer]",
			CooldownSeconds: 1800,
		},
	}

	section := ""
	videoListKey := ""
	lines := strings.Split(string(content), "\n")
	for idx, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if strings.HasSuffix(line, ":") && !strings.HasPrefix(line, "- ") {
			key := strings.TrimSuffix(line, ":")
			if section == "video" && (key == "formats" || key == "urls") {
				videoListKey = key
				if key == "formats" {
					cfg.Video.Formats = nil
				} else {
					cfg.Video.URLs = nil
				}
				continue
			}
			section = key
			videoListKey = ""
			continue
		}

		if strings.HasPrefix(line, "- ") {
			if section != "video" || videoListKey == "" {
				return Config{}, fmt.Errorf("line %d: unexpected list item", idx+1)
			}
			value := strings.TrimSpace(strings.Trim(strings.TrimPrefix(line, "- "), "\""))
			switch videoListKey {
			case "formats":
				cfg.Video.Formats = append(cfg.Video.Formats, value)
			case "urls":
				cfg.Video.URLs = append(cfg.Video.URLs, value)
			}
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			return Config{}, fmt.Errorf("line %d: invalid entry", idx+1)
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		value = strings.Trim(value, "\"")

		switch section {
		case "room":
			if key == "url" {
				cfg.Room.URL = value
			}
		case "video":
			switch key {
			case "source_dir":
				cfg.Video.SourceDir = value
			}
		case "stream":
			switch key {
			case "rtmp_url":
				cfg.Stream.RTMPURL = value
			case "stream_key":
				cfg.Stream.StreamKey = value
			case "ffmpeg_path":
				cfg.Stream.FFmpegPath = value
			case "ffmpeg_loglevel":
				cfg.Stream.FFmpegLogLevel = value
			case "copy_video":
				cfg.Stream.CopyVideo = parseBool(value)
			case "copy_audio":
				cfg.Stream.CopyAudio = parseBool(value)
			case "loop_single_input":
				cfg.Stream.LoopSingleInput = parseBool(value)
			}
		case "danmaku":
			switch key {
			case "enabled":
				cfg.Danmaku.Enabled = parseBool(value)
			case "command_prefix":
				cfg.Danmaku.CommandPrefix = value
			}
		case "notify":
			switch key {
			case "enabled":
				cfg.Notify.Enabled = parseBool(value)
			case "smtp_host":
				cfg.Notify.SMTPHost = value
			case "smtp_port":
				cfg.Notify.SMTPPort = value
			case "username":
				cfg.Notify.Username = value
			case "password":
				cfg.Notify.Password = value
			case "from":
				cfg.Notify.From = value
			case "to":
				cfg.Notify.To = splitList(value)
			case "subject_prefix":
				cfg.Notify.SubjectPrefix = value
			case "cooldown_seconds":
				cfg.Notify.CooldownSeconds = parseInt(value)
			}
		case "api":
			if key == "listen_addr" {
				cfg.API.ListenAddr = value
			}
		default:
			return Config{}, fmt.Errorf("line %d: unknown section %q", idx+1, section)
		}
	}

	if len(cfg.Video.Formats) == 0 {
		cfg.Video.Formats = []string{".ts", ".mp4"}
	}
	cfg.Video.URLs = compactStrings(cfg.Video.URLs)

	return cfg, nil
}

func applyEnv(cfg *Config) {
	overrideString(&cfg.Room.URL, "DOUYU_STREAMER_ROOM_URL")
	overrideString(&cfg.Video.SourceDir, "DOUYU_STREAMER_VIDEO_SOURCE_DIR")
	if value, ok := os.LookupEnv("DOUYU_STREAMER_VIDEO_URLS"); ok {
		cfg.Video.URLs = splitList(value)
	}
	overrideString(&cfg.Stream.RTMPURL, "DOUYU_STREAMER_STREAM_RTMP_URL")
	overrideString(&cfg.Stream.StreamKey, "DOUYU_STREAMER_STREAM_KEY")
	overrideString(&cfg.Stream.FFmpegPath, "DOUYU_STREAMER_FFMPEG_PATH")
	overrideString(&cfg.Stream.FFmpegLogLevel, "DOUYU_STREAMER_FFMPEG_LOGLEVEL")
	overrideString(&cfg.API.ListenAddr, "DOUYU_STREAMER_API_LISTEN_ADDR")
	overrideString(&cfg.Notify.SMTPHost, "DOUYU_STREAMER_NOTIFY_SMTP_HOST")
	overrideString(&cfg.Notify.SMTPPort, "DOUYU_STREAMER_NOTIFY_SMTP_PORT")
	overrideString(&cfg.Notify.Username, "DOUYU_STREAMER_NOTIFY_USERNAME")
	overrideString(&cfg.Notify.Password, "DOUYU_STREAMER_NOTIFY_PASSWORD")
	overrideString(&cfg.Notify.From, "DOUYU_STREAMER_NOTIFY_FROM")
	overrideString(&cfg.Notify.SubjectPrefix, "DOUYU_STREAMER_NOTIFY_SUBJECT_PREFIX")

	if value, ok := os.LookupEnv("DOUYU_STREAMER_DANMAKU_ENABLED"); ok {
		cfg.Danmaku.Enabled = parseBool(value)
	}
	overrideString(&cfg.Danmaku.CommandPrefix, "DOUYU_STREAMER_DANMAKU_COMMAND_PREFIX")
	if value, ok := os.LookupEnv("DOUYU_STREAMER_STREAM_COPY_VIDEO"); ok {
		cfg.Stream.CopyVideo = parseBool(value)
	}
	if value, ok := os.LookupEnv("DOUYU_STREAMER_STREAM_COPY_AUDIO"); ok {
		cfg.Stream.CopyAudio = parseBool(value)
	}
	if value, ok := os.LookupEnv("DOUYU_STREAMER_STREAM_LOOP_SINGLE_INPUT"); ok {
		cfg.Stream.LoopSingleInput = parseBool(value)
	}
	if value, ok := os.LookupEnv("DOUYU_STREAMER_NOTIFY_ENABLED"); ok {
		cfg.Notify.Enabled = parseBool(value)
	}
	if value, ok := os.LookupEnv("DOUYU_STREAMER_NOTIFY_TO"); ok {
		cfg.Notify.To = splitList(value)
	}
	if value, ok := os.LookupEnv("DOUYU_STREAMER_NOTIFY_COOLDOWN_SECONDS"); ok {
		cfg.Notify.CooldownSeconds = parseInt(value)
	}
	cfg.Notify.To = compactStrings(cfg.Notify.To)
}

func validate(cfg Config) error {
	switch {
	case cfg.Room.URL == "":
		return errors.New("room.url is required")
	case strings.TrimSpace(cfg.Video.SourceDir) == "" && len(cfg.Video.URLs) == 0:
		return errors.New("video.source_dir or video.urls is required")
	case cfg.Stream.RTMPURL == "":
		return errors.New("stream.rtmp_url is required")
	case cfg.Stream.StreamKey == "":
		return errors.New("stream.stream_key is required")
	case cfg.Stream.FFmpegPath == "":
		return errors.New("stream.ffmpeg_path is required")
	case cfg.Stream.FFmpegLogLevel == "":
		return errors.New("stream.ffmpeg_loglevel is required")
	case cfg.Notify.Enabled && cfg.Notify.SMTPHost == "":
		return errors.New("notify.smtp_host is required when notify.enabled=true")
	case cfg.Notify.Enabled && cfg.Notify.SMTPPort == "":
		return errors.New("notify.smtp_port is required when notify.enabled=true")
	case cfg.Notify.Enabled && cfg.Notify.From == "":
		return errors.New("notify.from is required when notify.enabled=true")
	case cfg.Notify.Enabled && len(cfg.Notify.To) == 0:
		return errors.New("notify.to is required when notify.enabled=true")
	}

	return nil
}

func overrideString(target *string, envKey string) {
	if value, ok := os.LookupEnv(envKey); ok && strings.TrimSpace(value) != "" {
		*target = value
	}
}

func parseBool(value string) bool {
	parsed, err := strconv.ParseBool(strings.TrimSpace(value))
	if err != nil {
		return false
	}
	return parsed
}

func parseInt(value string) int {
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return 0
	}
	return parsed
}

func splitList(value string) []string {
	fields := strings.FieldsFunc(value, func(r rune) bool {
		return r == ',' || r == '\n'
	})
	return compactStrings(fields)
}

func compactStrings(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(strings.Trim(value, "\""))
		if trimmed == "" {
			continue
		}
		result = append(result, trimmed)
	}
	return result
}
