package api

import (
	"github.com/1f349/violet/target"
)

type sourceJson struct {
	Src string `json:"src"`
}

func (s sourceJson) GetSource() string { return s.Src }

type routeSource target.RouteWithActive

func (r routeSource) GetSource() string { return r.Src }

type redirectSource target.RedirectWithActive

func (r redirectSource) GetSource() string { return r.Src }

var (
	_ sourceGetter = sourceJson{}
	_ sourceGetter = routeSource{}
	_ sourceGetter = redirectSource{}
)

type sourceGetter interface{ GetSource() string }
