package main

import "github.com/1f349/violet/utils"

type startUpConfig struct {
	SelfSigned    bool               `json:"self_signed"`
	ErrorPagePath string             `json:"error_page_path"`
	Listen        listenConfig       `json:"listen"`
	InkscapeCmd   string             `json:"inkscape"`
	RateLimit     uint64             `json:"rate_limit"`
	MetricsToken  string             `json:"metrics_token"`
	TableRefresh  utils.DurationText `json:"table_refresh"`
}

type listenConfig struct {
	Api   string `json:"api"`
	Http  string `json:"http"`
	Https string `json:"https"`
}
