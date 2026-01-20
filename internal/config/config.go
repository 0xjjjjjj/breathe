package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type JunkPattern struct {
	Name    string `yaml:"name"`
	Pattern string `yaml:"pattern"`
	Safe    bool   `yaml:"safe"`
}

type OrganizeRule struct {
	Match string `yaml:"match"`
	Dest  string `yaml:"dest"`
}

type Deletion struct {
	TrashThreshold string   `yaml:"trash_threshold"`
	AlwaysTrash    []string `yaml:"always_trash"`
}

type Config struct {
	JunkPatterns  []JunkPattern  `yaml:"junk_patterns"`
	OrganizeRules []OrganizeRule `yaml:"organize_rules"`
	Deletion      Deletion       `yaml:"deletion"`
}

func DefaultConfig() *Config {
	return &Config{
		JunkPatterns: []JunkPattern{
			{Name: "node_modules", Pattern: "**/node_modules", Safe: true},
			{Name: "JS build output", Pattern: "**/{dist,build,.next,.nuxt,out}/", Safe: true},
			{Name: "C# build output", Pattern: "**/{bin,obj}/", Safe: true},
			{Name: "Browser automation", Pattern: "**/{.chrome-data,chrome-data,puppeteer_data,.playwright}/", Safe: true},
			{Name: "Package caches", Pattern: "**/{.npm/_cacache,.yarn/cache,.pnpm-store}/", Safe: true},
			{Name: "Python cache", Pattern: "**/__pycache__/", Safe: true},
			{Name: "Git repos", Pattern: "**/.git", Safe: false},
		},
		OrganizeRules: []OrganizeRule{
			{Match: "*.{pdf,doc,docx,txt,rtf,odt}", Dest: "~/Documents/Downloads"},
			{Match: "*.{csv,xlsx,xls,json,xml}", Dest: "~/Documents/Data"},
			{Match: "*.{jpg,jpeg,png,gif,webp,svg,heic}", Dest: "~/Pictures/Downloads"},
			{Match: "Screenshot*.png", Dest: "~/Pictures/Screenshots"},
			{Match: "*.{dmg,pkg}", Dest: "~/Downloads/Installers"},
			{Match: "*.{zip,tar,gz,rar,7z}", Dest: "~/Downloads/Archives"},
			{Match: "*", Dest: "~/Downloads/Unsorted"},
		},
		Deletion: Deletion{
			TrashThreshold: "1GB",
			AlwaysTrash:    []string{".pdf", ".doc", ".xlsx"},
		},
	}
}

func Load(path string) (*Config, error) {
	if path == "" {
		return DefaultConfig(), nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultConfig(), nil
		}
		return nil, err
	}

	cfg := &Config{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

func DefaultPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "breathe", "config.yaml")
}

func DataPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", "breathe", "history.db")
}
