package cas9

import (
	"bytes"
	"strings"
	"syscall/js"

	"golang.org/x/net/html"
)

type vDOM struct {
	*html.Node
	JSNode *js.Value
	VParent, VFirstChild,
	VLastChild, VPrevSibling, VNextSibling *vDOM
	Events          []RegisteredEvent
	ParentComponent *Component
}

type VirtualDOMList []*vDOM

func (vd *vDOM) TagName() string {
	return vd.Node.Data
}

func (vd *vDOM) Delete() {
	if vd.VPrevSibling != nil && vd.VNextSibling != nil {
		vd.VPrevSibling.VNextSibling = vd.VNextSibling
		vd.VPrevSibling.NextSibling = vd.VNextSibling.Node
	} else if vd.VPrevSibling == nil && vd.VNextSibling == nil {
		vd.VParent.VFirstChild = nil
		vd.VParent.FirstChild = nil
	} else if vd.VNextSibling == nil {
		vd.VPrevSibling.VNextSibling = nil
		vd.VPrevSibling.NextSibling = nil
	} else if vd.VPrevSibling == nil {
		vd.VParent.VFirstChild = vd.VNextSibling
		vd.VParent.FirstChild = vd.VNextSibling.Node
		vd.VNextSibling.VPrevSibling = nil
		vd.VNextSibling.PrevSibling = nil
	}
	vd.JSNode.Call("remove")
	vd.JSNode = nil
}

// BuildDom generates new JSNodes, attaches them to tree with
// self as the root node. Returns the new root JSNode
func (vd *vDOM) BuildDOM() js.Value {
	var rootNode js.Value

	doc := js.Global().Get("document")

	if vd.Type == html.ElementNode {
		rootNode = doc.Call("createElement", vd.TagName())

		for _, a := range vd.Attr {
			rootNode.Call("setAttribute", a.Key, a.Val)
		}

		for _, e := range vd.Events {
			rootNode.Call("addEventListener", e.Name, js.NewCallback(func(v []js.Value) {
				e.Handler(EventObj{v[0]})
			}))
		}
	} else if vd.Type == html.TextNode {
		rootNode = doc.Call("createTextNode", vd.TagName())
	}

	vd.JSNode = &rootNode

	for c := vd.VFirstChild; c != nil; c = c.VNextSibling {
		jsVal := c.BuildDOM()
		c.JSNode = &jsVal
		vd.JSNode.Call("appendChild", jsVal)
	}

	return rootNode

}

func (vd *vDOM) Content() string {
	tn := vd.FirstChild
	if tn != nil && tn.Type == html.TextNode {
		return strings.TrimSpace(tn.Data)
	}
	return ""
}

func findNode(tagName string, n *vDOM) *vDOM {
	if n.Type == html.ElementNode && n.Data == tagName {
		return n
	}
	for c := n.VFirstChild; c != nil; c = c.VNextSibling {
		child := findNode(tagName, c)
		if child != nil {
			return child
		}
	}
	return nil
}

func findAttribute(prefix string, n *vDOM) (*vDOM, html.Attribute) {
	if n.Type == html.ElementNode {
		for _, a := range n.Attr {
			if strings.HasPrefix(a.Key, prefix) {
				return n, a
			}
		}
	}
	for c := n.VFirstChild; c != nil; c = c.VNextSibling {
		child, attr := findAttribute(prefix, c)
		if child != nil {
			return child, attr
		}
	}
	return nil, html.Attribute{}
}

func TrimTo(tagName string, startNode *vDOM) VirtualDOMList {

	var vdl = make(VirtualDOMList, 0, 10)

	found := findNode(tagName, startNode)
	if found != nil {
		for c := found.VFirstChild; c != nil; c = c.VNextSibling {
			vdl = append(vdl, c)
		}
	}

	return vdl
}

func spliceTree(subVDom VirtualDOMList, componentNode *vDOM) {
	if len(subVDom) > 0 {

		first := subVDom[0]
		last := subVDom[len(subVDom)-1]

		before := componentNode.VPrevSibling
		after := componentNode.VNextSibling

		if before != nil && after != nil {
			before.VNextSibling = first
			before.NextSibling = first.Node
			first.VPrevSibling = before
			first.PrevSibling = before.Node

			after.VPrevSibling = last
			after.PrevSibling = last.Node
			last.VNextSibling = after
			last.NextSibling = after.Node
		} else if before == nil && after == nil {
			first.VParent = componentNode.VParent
			first.Parent = componentNode.VParent.Node
			componentNode.VParent.VFirstChild = first
			componentNode.VParent.FirstChild = first.Node
		} else if before == nil {
			first.VParent = componentNode.VParent
			first.Parent = componentNode.VParent.Node
			componentNode.VParent.VFirstChild = first
			componentNode.VParent.FirstChild = first.Node

			last.VNextSibling = after
			last.NextSibling = after.Node
			after.VPrevSibling = last
			after.PrevSibling = last.Node
		} else if after == nil {
			first.VPrevSibling = before
			first.PrevSibling = before.Node
			before.VNextSibling = before
			before.NextSibling = before.Node

			componentNode.VParent.VLastChild = last
			componentNode.VParent.LastChild = last.Node
		}

		setParent(subVDom, componentNode.VParent)
	}
}

func setParent(subVDom VirtualDOMList, parent *vDOM) {
	for _, i := range subVDom {
		i.VParent = parent
		i.Parent = parent.Node
	}
}

func rendervDOM(node *vDOM) string {
	var buf bytes.Buffer
	render(&buf, node)
	return buf.String()
}
