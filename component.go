package cas9

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"reflect"
	"strings"

	"github.com/google/uuid"
	"golang.org/x/net/html"
)

type ComponentView interface {
	Render() string

	PreProcessTemplate() string
	Refresh()
	RegisterComponent(ComponentView) string
	GenerateNewvDOM() VirtualDOMList
	DeleteDOM()
	Init(ComponentView)
	GetParent() ComponentView
	GetVDList() VirtualDOMList
}

type Component struct {
	Parent             ComponentView
	templateCache      *template.Template
	renderedComponents map[string]ComponentView
	VDList             VirtualDOMList
	renderedEvents     map[string]RegisteredEvent
	renderedBindings   map[string]RegisteredBinding
}

type RegisteredBinding struct {
	FieldName string
	FieldVal  reflect.Value
}

type RegisteredEvent struct {
	Name    string
	Handler func(EventObj)
}

func (c *Component) Init(p ComponentView) {
	c.Parent = p

	baseVal := reflect.ValueOf(p).Elem()
	for i := 0; i < baseVal.NumField(); i++ {
		field := baseVal.Field(i)

		if field.Kind() == reflect.Slice {
			for n := 0; n < field.Len(); n++ {
				f := field.Index(n)
				if comp, ok := f.Interface().(ComponentView); ok {
					if comp.GetParent() == nil {
						comp.Init(comp)
					}
				}
			}
		} else {
			// does implement ComponentView?
			if comp, ok := field.Interface().(ComponentView); ok {
				if comp.GetParent() == nil {
					comp.Init(comp)
				}
			}
		}
	}

}

func (c *Component) Set(key string, val interface{}) {
	comp := reflect.ValueOf(c.Parent).Elem()
	comp.FieldByName(key).Set(reflect.ValueOf(val))
	c.Refresh()
}

func (c *Component) GetParent() ComponentView {
	return c.Parent
}

func (c *Component) GetVDList() VirtualDOMList {
	return c.VDList
}

func (c *Component) DeleteDOM() {
	for _, dom := range c.VDList {
		dom.Delete()
	}
}

func (c *Component) RegisterComponent(comp ComponentView) string {
	key := uuid.New().String()
	c.renderedComponents[key] = comp
	return fmt.Sprintf("<cas9-component>%v</cas9-component>", key)
}

func (c *Component) RegisterEvent(eventName, handler string) template.HTMLAttr {
	key := uuid.New().String()
	method := reflect.ValueOf(c.Parent).MethodByName(handler).Interface().(func(EventObj))
	c.renderedEvents[key] = RegisteredEvent{eventName, method}
	return template.HTMLAttr(fmt.Sprintf("cas9-event-%v=\"%v\"", len(c.renderedEvents), key))
}

func (c *Component) render(all interface{}) template.HTML {
	var sb strings.Builder

	switch reflect.TypeOf(all).Kind() {
	case reflect.Slice:
		s := reflect.ValueOf(all)

		if s.Len() > 0 {
			_, ok := s.Index(0).Interface().(ComponentView)
			if ok {
				for i := 0; i < s.Len(); i++ {
					sb.WriteString(c.RegisterComponent(s.Index(i).Interface().(ComponentView)))
				}
			} else {
				panic("Couldn't render, not a ComponentView")
			}
		}
	default:
		comp, ok := all.(ComponentView)
		if ok {
			sb.WriteString(c.RegisterComponent(comp))
		} else {
			panic("Couldn't render, not a ComponentView")
		}
	}

	return template.HTML(sb.String())
}

func (c *Component) RegisterEvents(em SelectorEvents) {

}

func (c *Component) ProcessComponents(vd VirtualDOMList) {

	for _, v := range vd {
		for componentNode, attr := findAttribute("cas9-binding", v); componentNode != nil; componentNode, attr = findAttribute("cas9-binding", v) {
			for i, a := range componentNode.Attr {
				if a.Key == attr.Key && a.Val == attr.Val {
					componentNode.Events = append(componentNode.Events, RegisteredEvent{Name: "change", Handler: func(e EventObj) {
						binding := c.renderedBindings[a.Val]
						binding.FieldVal.Set(reflect.ValueOf(e.Get("target").Get("value").String()))

						println("got here")

					}})
					componentNode.Attr = append(componentNode.Attr[:i], componentNode.Attr[i+1:]...)
					break
				}
			}
		}
		for componentNode, attr := findAttribute("cas9-event", v); componentNode != nil; componentNode, attr = findAttribute("cas9-event", v) {
			for i, a := range componentNode.Attr {
				if a.Key == attr.Key && a.Val == attr.Val {
					componentNode.Events = append(componentNode.Events, c.renderedEvents[a.Val])
					componentNode.Attr = append(componentNode.Attr[:i], componentNode.Attr[i+1:]...)
					break
				}
			}
		}
		for componentNode := findNode("cas9-component", v); componentNode != nil; componentNode = findNode("cas9-component", v) {
			subVDom := c.renderedComponents[componentNode.Content()].GenerateNewvDOM()
			spliceTree(subVDom, componentNode)
		}
	}

}

// Refresh Entire new JSNode tree is created and replace root node
// todo: only update(new, update, delete) changed nodes
func (c *Component) Refresh() {
	oldList := c.VDList
	c.GenerateNewvDOM()
	for i, dom := range c.VDList {
		dom.VParent.JSNode = oldList[i].VParent.JSNode
		jsVal := dom.BuildDOM()
		oldNode := *oldList[i].JSNode
		oldNode.Call("replaceWith", jsVal)
	}
}

func (c *Component) GenerateNewvDOM() VirtualDOMList {
	c.Init(c.Parent)

	tplStr := c.PreProcessTemplate()

	doc, err := html.Parse(strings.NewReader(tplStr))
	if err != nil {
		log.Fatal(err)
	}

	virtualDOM := TrimTo("body", c.Wrap(doc))

	c.ProcessComponents(virtualDOM)

	c.VDList = virtualDOM

	return virtualDOM
}

func (c *Component) Wrap(node *html.Node) *vDOM {

	if node == nil {
		return nil
	}

	newNode := &vDOM{Node: node, ParentComponent: c}
	newNode.VFirstChild = c.Wrap(node.FirstChild)

	if newNode.VFirstChild != nil {
		newNode.VFirstChild.VParent = newNode

		ref := newNode.VFirstChild
		for cn := newNode.VFirstChild.NextSibling; cn != nil; cn = cn.NextSibling {
			ref.VNextSibling = c.Wrap(cn)
			ref.VNextSibling.VParent = newNode
			ref.VNextSibling.VPrevSibling = ref

			ref = ref.VNextSibling
		}

		newNode.VLastChild = ref
	}

	return newNode
}

func (c *Component) RegisterBinding(fieldName string) template.HTMLAttr {
	key := uuid.New().String()
	field := reflect.ValueOf(c.Parent).Elem().FieldByName(fieldName)
	c.renderedBindings[key] = RegisteredBinding{FieldName: fieldName, FieldVal: field}

	return template.HTMLAttr(fmt.Sprintf("cas9-binding-%v=\"%v\"", len(c.renderedBindings), key))
}

func (c *Component) PreProcessTemplate() string {
	c.renderedComponents = make(map[string]ComponentView)
	c.renderedEvents = make(map[string]RegisteredEvent)
	c.renderedBindings = make(map[string]RegisteredBinding)

	t := template.Must(template.New("mainComp").Funcs(template.FuncMap{
		"render": c.render,
		"bind": func(fieldName string) template.HTMLAttr {
			return c.RegisterBinding(fieldName)
		},
		"on": func(eventName, handler string) template.HTMLAttr {
			return c.RegisterEvent(eventName, handler)
		},
		"classIf": func(className string, show bool) string {
			if show {
				return " " + strings.TrimSpace(className)
			}
			return ""
		},
	}).Parse(strings.TrimSpace(c.Parent.Render())))

	var tpl bytes.Buffer

	if err := t.Execute(&tpl, c.Parent); err != nil {
		println(err.Error())
	}

	return tpl.String()
}
