package cas9

import (
	"reflect"
	"syscall/js"
	"time"
)

func AttachTo(selector string, c ComponentView) {
	head := js.Global().Get("document").Get("head")
	mainNode := js.Global().Get("document").Call("querySelector", selector)

	now := time.Now()

	c.Init(c)

	dom := c.GenerateNewvDOM()

	styles := c.GetStyles()
	for _, style := range styles {
		head.Call("appendChild", style.BuildDOM())
	}

	for _, d := range dom {
		//d.VParent.JSNode = &mainNode
		mainNode.Call("appendChild", d.BuildDOM())
	}

	println("render took: " + time.Now().Sub(now).String())
}

func (c *Component) Append(slcPtr, iPtr interface{}) {
	slice := reflect.ValueOf(slcPtr)
	item := reflect.ValueOf(iPtr)

	sliceVal := slice.Elem()
	if slice.Kind() != reflect.Ptr {
		panic("Non-pointer passed to RemoveSliceItem")
	} else if sliceVal.Kind() != reflect.Slice {
		panic("Non-slice passed to RemoveSliceItem")
	}

	if item.Kind() != reflect.Ptr {
		panic("Non-pointer passed to RemoveSliceItem")
	}

	if comp, ok := item.Interface().(ComponentView); ok {
		comp.Init(comp)

		if cLen := sliceVal.Len(); cLen > 0 {
			lastOldComp := sliceVal.Index(cLen - 1).Interface().(ComponentView)

			vdl := lastOldComp.GetVDList()
			if vLen := len(vdl); vLen > 0 {
				one := vdl[vLen-1]
				two := one.VNextSibling

				insertList := comp.GenerateNewvDOM()

				if len(insertList) > 0 {
					first := insertList[0]
					last := insertList[len(insertList)-1]

					one.VNextSibling = first
					one.NextSibling = first.Node
					first.VPrevSibling = one
					first.PrevSibling = one.Node

					if two != nil {
						two.VPrevSibling = last
						two.PrevSibling = last.Node
						last.VNextSibling = two
						last.NextSibling = two.Node
					}
				}

				newList := comp.GetVDList()

				setParent(newList, one.VParent)

				var nextSibNode js.Value
				if two != nil {
					nextSibNode = *two.JSNode
				} else {
					nextSibNode = js.Null()
				}

				for _, node := range newList {
					jsNode := node.BuildDOM()
					one.VParent.JSNode.Call("insertBefore", jsNode, nextSibNode)
				}
				sliceVal.Set(reflect.Append(sliceVal, item))
			} else {
				sliceVal.Set(reflect.Append(sliceVal, item))
				c.Refresh()
			}
		} else {
			sliceVal.Set(reflect.Append(sliceVal, item))
			c.Refresh()
		}

	} else {
		panic("Can't append, iPtr isn't a ComponentView")
	}

}

func RemoveSliceItem(slcPtr, iPtr interface{}) {

	slice := reflect.ValueOf(slcPtr)
	item := reflect.ValueOf(iPtr)

	sliceVal := slice.Elem()

	if slice.Kind() != reflect.Ptr {
		panic("Non-pointer passed to RemoveSliceItem")
	} else if sliceVal.Kind() != reflect.Slice {
		panic("Non-slice passed to RemoveSliceItem")
	}

	if item.Kind() != reflect.Ptr {
		panic("Non-pointer passed to RemoveSliceItem")
	}

	for i := 0; i < sliceVal.Len(); i++ {
		if sliceVal.Index(i).Pointer() == item.Pointer() {
			a := sliceVal.Slice(0, i)
			b := sliceVal.Slice(i+1, sliceVal.Len())
			sliceVal.Set(reflect.AppendSlice(a, b))

			if comp, ok := item.Interface().(ComponentView); ok {
				comp.DeleteDOM()
			} else {
				panic("Can't delete DOM, iPtr isn't a ComponentView")
			}

			break
		}
	}

}
