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
	API     APIConfig
}

type RoomConfig struct {
	URL string
}

type VideoConfig struct {
	SourceDir string
	Formats   []string
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
	}

	section := ""
	inVideoFormats := false
	lines := strings.Split(string(content), "\n")
	for idx, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if section == "video" && line == "formats:" {
			cfg.Video.Formats = nil
			inVideoFormats = true
			continue
		}

		if strings.HasSuffix(line, ":") && !strings.HasPrefix(line, "- ") {
			section = strings.TrimSuffix(line, ":")
			inVideoFormats = false
			continue
		}

		if strings.HasPrefix(line, "- ") {
			if !inVideoFormats {
				return Config{}, fmt.Errorf("line %d: unexpected list item", idx+1)
			}
			cfg.Video.Formats = append(cfg.Video.Formats, strings.TrimSpace(strings.TrimPrefix(line, "- ")))
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

	return cfg, nil
}

func applyEnv(cfg *Config) {
	overrideString(&cfg.Room.URL, "DOUYU_STREAMER_ROOM_URL")
	overrideString(&cfg.Video.SourceDir, "DOUYU_STREAMER_VIDEO_SOURCE_DIR")
	overrideString(&cfg.Stream.RTMPURL, "DOUYU_STREAMER_STREAM_RTMP_URL")
	overrideString(&cfg.Stream.StreamKey, "DOUYU_STREAMER_STREAM_KEY")
	overrideString(&cfg.Stream.FFmpegPath, "DOUYU_STREAMER_FFMPEG_PATH")
	overrideString(&cfg.Stream.FFmpegLogLevel, "DOUYU_STREAMER_FFMPEG_LOGLEVEL")
	overrideString(&cfg.API.ListenAddr, "DOUYU_STREAMER_API_LISTEN_ADDR")

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
}

func validate(cfg Config) error {
	switch {
	case cfg.Room.URL == "":
		return errors.New("room.url is required")
	case cfg.Video.SourceDir == "":
		return errors.New("video.source_dir is required")
	case cfg.Stream.RTMPURL == "":
		return errors.New("stream.rtmp_url is required")
	case cfg.Stream.StreamKey == "":
		return errors.New("stream.stream_key is required")
	case cfg.Stream.FFmpegPath == "":
		return errors.New("stream.ffmpeg_path is required")
	case cfg.Stream.FFmpegLogLevel == "":
		return errors.New("stream.ffmpeg_loglevel is required")
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
