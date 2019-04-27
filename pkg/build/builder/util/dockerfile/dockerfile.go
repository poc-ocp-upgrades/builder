package dockerfile

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"github.com/docker/docker/builder/dockerfile/command"
	"github.com/docker/docker/builder/dockerfile/parser"
)

func Parse(r io.Reader) (*parser.Node, error) {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	result, err := parser.Parse(r)
	if err != nil {
		return nil, err
	}
	return result.AST, nil
}
func Write(node *parser.Node) []byte {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	if node == nil {
		return nil
	}
	buf := &bytes.Buffer{}
	if len(node.Value) > 0 {
		buf.Write([]byte(strings.ToUpper(node.Value)))
		for _, flag := range node.Flags {
			buf.Write([]byte(" "))
			buf.Write([]byte(flag))
		}
		switch node.Value {
		case command.Onbuild:
			if node.Next != nil && len(node.Next.Children) > 0 {
				buf.Write([]byte(" "))
				buf.Write(Write(node.Next.Children[0]))
			}
			return buf.Bytes()
		case command.Env, command.Label:
			for n := node.Next; n != nil; n = n.Next {
				if buf.Len() > 0 {
					buf.Write([]byte(" "))
				}
				buf.Write([]byte(n.Value))
				buf.Write([]byte("="))
				if n.Next != nil {
					buf.Write([]byte(n.Next.Value))
				}
				n = n.Next
			}
			buf.Write([]byte("\n"))
			return buf.Bytes()
		default:
			if node.Attributes["json"] {
				var values []string
				for n := node.Next; n != nil; n = n.Next {
					values = append(values, n.Value)
				}
				out, _ := json.Marshal(values)
				buf.Write([]byte(" "))
				buf.Write(out)
				buf.Write([]byte("\n"))
				return buf.Bytes()
			}
			for n := node.Next; n != nil; n = n.Next {
				if buf.Len() > 0 {
					buf.Write([]byte(" "))
				}
				buf.Write([]byte(n.Value))
			}
			buf.Write([]byte("\n"))
			return buf.Bytes()
		}
	}
	for _, child := range node.Children {
		buf.Write(Write(child))
	}
	return buf.Bytes()
}
func FindAll(node *parser.Node, cmd string) []int {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	if node == nil {
		return nil
	}
	var indices []int
	for i, child := range node.Children {
		if child != nil && child.Value == cmd {
			indices = append(indices, i)
		}
	}
	return indices
}
func InsertInstructions(node *parser.Node, pos int, instructions string) error {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	if node == nil {
		return fmt.Errorf("cannot insert instructions in a nil node")
	}
	if pos < 0 || pos > len(node.Children) {
		return fmt.Errorf("pos %d out of range [0, %d]", pos, len(node.Children)-1)
	}
	newChild, err := Parse(strings.NewReader(instructions))
	if err != nil {
		return err
	}
	node.Children = append(node.Children[:pos], append(newChild.Children, node.Children[pos:]...)...)
	return nil
}
func baseImages(node *parser.Node) []string {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	var images []string
	for _, pos := range FindAll(node, command.From) {
		images = append(images, nextValues(node.Children[pos])...)
	}
	return images
}
func exposedPorts(node *parser.Node) [][]string {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	var allPorts [][]string
	var ports []string
	froms := FindAll(node, command.From)
	exposes := FindAll(node, command.Expose)
	for i, j := len(froms)-1, len(exposes)-1; i >= 0; i-- {
		for ; j >= 0 && exposes[j] > froms[i]; j-- {
			ports = append(nextValues(node.Children[exposes[j]]), ports...)
		}
		allPorts = append([][]string{ports}, allPorts...)
		ports = nil
	}
	return allPorts
}
func nextValues(node *parser.Node) []string {
	_logClusterCodePath()
	defer _logClusterCodePath()
	_logClusterCodePath()
	defer _logClusterCodePath()
	if node == nil {
		return nil
	}
	var values []string
	for next := node.Next; next != nil; next = next.Next {
		values = append(values, next.Value)
	}
	return values
}
