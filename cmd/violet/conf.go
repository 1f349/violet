package main

type startUpConfig struct {
	Database      string       `json:"db"`
	MjwtPubKey    string       `json:"mjwt_pub_key"`
	CertPath      string       `json:"cert_path"`
	KeyPath       string       `json:"key_path"`
	SelfSigned    bool         `json:"self_signed"`
	ErrorPagePath string       `json:"error_page_path"`
	Listen        listenConfig `json:"listen"`
	InkscapeCmd   string       `json:"inkscape"`
	RateLimit     uint64       `json:"rate_limit"`
}

type listenConfig struct {
	Api   string `json:"api"`
	Http  string `json:"http"`
	Https string `json:"https"`
}
