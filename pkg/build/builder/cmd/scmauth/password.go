package scmauth

import (
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	s2igit "github.com/openshift/source-to-image/pkg/scm/git"
	builder "github.com/openshift/builder/pkg/build/builder"
)

const (
	DefaultUsername		= "builder"
	UsernamePasswordName	= "password"
	UsernameSecret		= "username"
	PasswordSecret		= "password"
	TokenSecret		= "token"
	UserPassGitConfig	= `# credential git config
[credential]
   helper = store --file=%s
`
)

type UsernamePassword struct{ SourceURL s2igit.URL }

func (u UsernamePassword) Setup(baseDir string, context SCMAuthContext) error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	if !(u.SourceURL.Type == s2igit.URLTypeURL && (u.SourceURL.URL.Scheme == "http" || u.SourceURL.URL.Scheme == "https") && u.SourceURL.URL.Opaque == "") {
		return nil
	}
	usernameSecret, err := readSecret(baseDir, UsernameSecret)
	if err != nil {
		return err
	}
	passwordSecret, err := readSecret(baseDir, PasswordSecret)
	if err != nil {
		return err
	}
	tokenSecret, err := readSecret(baseDir, TokenSecret)
	if err != nil {
		return err
	}
	overrideSourceURL, gitconfigURL, err := doSetup(u.SourceURL.URL, usernameSecret, passwordSecret, tokenSecret)
	if err != nil {
		return err
	}
	if overrideSourceURL != nil {
		if err := context.SetOverrideURL(overrideSourceURL); err != nil {
			return err
		}
	}
	if gitconfigURL != nil {
		gitcredentials, err := ioutil.TempFile("", "gitcredentials.")
		if err != nil {
			return err
		}
		defer gitcredentials.Close()
		gitconfig, err := ioutil.TempFile("", "gitcredentialscfg.")
		if err != nil {
			return err
		}
		defer gitconfig.Close()
		configContent := fmt.Sprintf(UserPassGitConfig, gitcredentials.Name())
		glog.V(5).Infof("Adding username/password credentials to git config:\n%s\n", configContent)
		fmt.Fprintf(gitconfig, "%s", configContent)
		fmt.Fprintf(gitcredentials, "%s", gitconfigURL.String())
		return ensureGitConfigIncludes(gitconfig.Name(), context)
	}
	return nil
}
func doSetup(sourceURL url.URL, usernameSecret, passwordSecret, tokenSecret string) (*url.URL, *url.URL, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	urlUsername := ""
	urlPassword := ""
	if sourceURL.User != nil {
		urlUsername = sourceURL.User.Username()
		urlPassword, _ = sourceURL.User.Password()
	}
	username := usernameSecret
	if username == "" {
		username = urlUsername
	}
	password := tokenSecret
	if password == "" {
		password = passwordSecret
	}
	if password == "" {
		password = urlPassword
	}
	if password == "" && username == urlUsername {
		return nil, nil, nil
	}
	if username == "" {
		username = DefaultUsername
	}
	overrideSourceURL := sourceURL
	overrideSourceURL.User = nil
	configURL := sourceURL
	configURL.User = url.UserPassword(username, password)
	return &overrideSourceURL, &configURL, nil
}
func (_ UsernamePassword) Name() string {
	_logClusterCodePath()
	defer _logClusterCodePath()
	return UsernamePasswordName
}
func (_ UsernamePassword) Handles(name string) bool {
	_logClusterCodePath()
	defer _logClusterCodePath()
	switch name {
	case UsernameSecret, PasswordSecret, TokenSecret:
		return true
	}
	return false
}
func readSecret(baseDir, fileName string) (string, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	path := filepath.Join(baseDir, fileName)
	lines, err := builder.ReadLines(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	if len(lines) == 0 {
		return "", nil
	}
	return lines[0], nil
}
