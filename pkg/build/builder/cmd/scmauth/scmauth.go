package scmauth

import "net/url"

type SCMAuth interface {
	Name() string
	Handles(name string) bool
	Setup(baseDir string, context SCMAuthContext) error
}
type SCMAuthContext interface {
	Get(name string) (string, bool)
	Set(name, value string) error
	SetOverrideURL(u *url.URL) error
}
