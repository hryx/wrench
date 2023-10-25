package zon

import (
	"fmt"
	"io"
)

const (
	stateStart         = "start"
	stateDot           = "dot"
	stateValue         = "value"
	stateNextValue     = "next-value"
	stateTag           = "tag"
	stateStringLiteral = "string-literal"
	stateStartComment  = "start-comment"
	stateComment       = "comment"
)

func Parse(contents string) (*Node, error) {
	var (
		nextState = stateStart
		prevState = stateStart
		line      = 0
		column    = 0
		tree      *Node
		stack     []*Node
		stackName []string
		tagName   string
		stringLit string
		runes     = []rune(contents)
	)
	for i := 0; i < len(runes); i++ {
		c := runes[i]
		column++
		if c == '\n' {
			line++
			column = 0
		}
		expect := func(expected rune, state string) error {
			if c != expected {
				return fmt.Errorf("%v:%v: expected %s (%q), found %q", line, column, state, string(expected), string(c))
			}
			return nil
		}
		// fmt.Printf("%v:%v: %s %s - %s\n", line, column, string(c), nextState, stackName)
		switch nextState {
		case stateStart:
			if c == ' ' || c == '\n' {

			} else if c == '/' {
				prevState = stateStart
				nextState = stateStartComment
			} else {
				if err := expect('.', nextState); err != nil {
					return nil, err
				}
				nextState = stateDot
			}
		case stateStartComment:
			if err := expect('/', nextState); err != nil {
				return nil, err
			}
			if err := expect('/', nextState); err != nil {
				return nil, err
			}
			nextState = stateComment
		case stateComment:
			if c == '\n' {
				nextState = prevState
			}
		case stateDot:
			if c == '{' {
				if err := expect('{', nextState); err != nil {
					return nil, err
				}
				nextState = stateValue
				if tree == nil {
					stack = append(stack, &Node{})
					stackName = append(stackName, "root")
					tree = stack[len(stack)-1]
				}
			} else {
				tagName += string(c)
				nextState = stateTag
			}
		case stateValue:
			if c == ' ' || c == '\n' {

			} else if c == '"' {
				if err := expect('"', nextState); err != nil {
					return nil, err
				}
				nextState = stateStringLiteral
			} else if c == '}' {
				// object close
				stack = stack[:len(stack)-1]
				stackName = stackName[:len(stackName)-1]
				nextState = stateNextValue
			} else {
				if err := expect('.', nextState); err != nil {
					return nil, err
				}
				nextState = stateDot
			}
		case stateNextValue:
			if c == ' ' || c == '\n' {

			} else if c == '}' {
				// object close
				stack = stack[:len(stack)-1]
				stackName = stackName[:len(stackName)-1]
				nextState = stateNextValue
				continue
			} else {
				if err := expect(',', nextState); err != nil {
					return nil, err
				}
				nextState = stateValue
			}
		case stateTag:
			if c == ' ' || c == '\n' {

			} else if c == '=' {
				parent := stack[len(stack)-1]
				parent.Tags = append(parent.Tags, Tag{Name: tagName, Node: Node{}})
				stack = append(stack, &parent.Tags[len(parent.Tags)-1].Node)
				stackName = append(stackName, tagName)
				nextState = stateValue
				tagName = ""
				continue
			} else {
				tagName += string(c)
			}
		case stateStringLiteral:
			if c == '"' {
				stack[len(stack)-1].StringLiteral = stringLit
				stack = stack[:len(stack)-1]
				stackName = stackName[:len(stackName)-1]
				tagName = ""
				stringLit = ""
				nextState = stateNextValue
			} else {
				stringLit += string(c)
			}
		}
	}
	if len(stack) != 0 {
		fmt.Println(len(stack), stackName)
		panic("unexpected: stack not emptied")
	}
	if tree == nil {
		return &Node{}, nil
	}
	return tree, nil
}

type Tag struct {
	Name string
	Node Node
}

type Node struct {
	Tags          []Tag
	StringLiteral string
}

func (n *Node) Write(w io.Writer, indent, prefix string) error {
	if err := n.write(w, indent, prefix); err != nil {
		return err
	}
	fmt.Fprintf(w, "\n")
	return nil
}

func (n *Node) write(w io.Writer, indent, prefix string) error {
	if n.StringLiteral != "" {
		fmt.Fprintf(w, "%q", n.StringLiteral)
		return nil
	}
	fmt.Fprintf(w, ".{\n")
	for _, tag := range n.Tags {
		fmt.Fprintf(w, prefix+indent+".%s = ", tag.Name)
		_ = tag.Node.write(w, indent, prefix+indent)
		fmt.Fprintf(w, ",")
		fmt.Fprintf(w, "\n")
	}
	fmt.Fprintf(w, prefix+"}")
	return nil
}

func (n *Node) Child(tagName string) *Node {
	for i, tag := range n.Tags {
		if tag.Name == tagName {
			return &n.Tags[i].Node
		}
	}
	return nil
}
