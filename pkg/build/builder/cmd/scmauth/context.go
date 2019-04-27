package scmauth

import (
	"fmt"
	"net/url"
)

type defaultSCMContext struct {
	vars		map[string]string
	overrideURL	*url.URL
}

func NewDefaultSCMContext() *defaultSCMContext {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	return &defaultSCMContext{vars: make(map[string]string)}
}
func (c *defaultSCMContext) Get(name string) (string, bool) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	value, ok := c.vars[name]
	return value, ok
}
func (c *defaultSCMContext) Set(name, value string) error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	if oldValue, isSet := c.Get(name); isSet && value != oldValue {
		return fmt.Errorf("cannot set the value of variable %s with value %q. Existing value: %q", name, value, oldValue)
	}
	c.vars[name] = value
	return nil
}
func (c *defaultSCMContext) SetOverrideURL(u *url.URL) error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	if c.overrideURL != nil && c.overrideURL.String() != u.String() {
		return fmt.Errorf("cannot set the value of overrideURL with value %s. Existing value: %s", c.overrideURL.String(), u.String())
	}
	c.overrideURL = u
	return nil
}
func (c *defaultSCMContext) OverrideURL() *url.URL {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	return c.overrideURL
}
func (c *defaultSCMContext) Env() []string {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	env := []string{}
	for k, v := range c.vars {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}
	return env
}
