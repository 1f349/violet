package api

import (
	"github.com/MrMelon54/violet/target"
)

type sourceJson struct {
	Src string `json:"src"`
}

func (s sourceJson) GetSource() string { return s.Src }

type routeSource target.Route

func (r routeSource) GetSource() string { return r.Src }

type redirectSource target.Redirect

func (r redirectSource) GetSource() string { return r.Src }

var (
	_ sourceGetter = sourceJson{}
	_ sourceGetter = routeSource{}
	_ sourceGetter = redirectSource{}
)

type sourceGetter interface{ GetSource() string }
