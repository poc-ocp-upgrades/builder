package dockerfile

import (
	"encoding/json"
	"fmt"
	"strings"
	"github.com/docker/docker/builder/dockerfile/command"
)

type KeyValue struct {
	Key	string
	Value	string
}

func Env(m []KeyValue) (string, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	return keyValueInstruction(command.Env, m)
}
func From(image string) (string, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	return unquotedArgsInstruction(command.From, image)
}
func Label(m []KeyValue) (string, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	return keyValueInstruction(command.Label, m)
}
func Run(cmd string) (string, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	return unquotedArgsInstruction(command.Run, cmd)
}
func keyValueInstruction(cmd string, m []KeyValue) (string, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	s := []string{strings.ToUpper(cmd)}
	for _, kv := range m {
		k, err := json.Marshal(kv.Key)
		if err != nil {
			return "", err
		}
		v, err := json.Marshal(kv.Value)
		if err != nil {
			return "", err
		}
		s = append(s, fmt.Sprintf("%s=%s", k, v))
	}
	return strings.Join(s, " "), nil
}
func unquotedArgsInstruction(cmd string, args ...string) (string, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	s := []string{strings.ToUpper(cmd)}
	for _, arg := range args {
		s = append(s, strings.Split(arg, "\n")...)
	}
	return strings.TrimRight(strings.Join(s, " "), " "), nil
}
