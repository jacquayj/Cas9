# Cas9: A Client-Side Web Library for Go

## Build Instructions

* Install golang >= go1.11
* Install goexec: `go get -u github.com/shurcooL/goexec`

### Create HTML Page

```html
<html>
	<head>
		<meta charset="utf-8">
		<script src="wasm_exec.js"></script>
		<script>
			const go = new Go();
            let mod, inst;
            WebAssembly.instantiateStreaming(fetch("main.wasm"), go.importObject).then(async (result) => {
                mod = result.module;
                inst = result.instance;
                await go.run(inst)
            });
		</script>
	</head>
	<body id="app"></body>
</html>
```

### Write your application

```go
package main

import "github.com/jjacquay712/cas9"

type TodoItem struct {
	cas9.Component

	Parent *TodoList

	Title, Content string
	IsDone         bool
	IsActive       bool
}

func (ti *TodoItem) OnClick(e cas9.EventObj) {
	ti.IsActive = !ti.IsActive
}

func (ti *TodoItem) Delete(e cas9.EventObj) {
	cas9.RemoveSliceItem(&ti.Parent.Items, ti)
}

func (ti *TodoItem) Render() string {
	return `
	<li class="todo-item{{ classIf "active" .IsActive }}" {{ on "click" "OnClick" }}>
		<strong>{{ .Title }}</strong>
		<br />
		{{ .Content }}
		<br />
		<input type="button" value="Delete" {{ on "click" "Delete" }} />
	</li>`
}

type TodoItems []*TodoItem

type TodoList struct {
	cas9.Component

	NewItemTitle, NewItemCont string

	Items TodoItems
}

func (tl *TodoList) CreateNewTodo(e cas9.EventObj) {
	tl.Append(&tl.Items, NewTodoItem(tl.NewItemTitle, tl.NewItemCont, tl))
}

func (tl *TodoList) Render() string {
	return `
	<div class="todo-list">
		<h2>My TODO List</h2>

		<hr />

		<ul>
			{{ render .Items }}
		</ul>

		<hr />
		
		<input type="text" {{ bind "NewItemTitle" }} />
		<input type="text" {{ bind "NewItemCont" }} />
		
		<input type="button" value="createNewTodo" {{ on "click" "CreateNewTodo" }} />
	</div>`
}

func NewTodoItem(title, content string, parent *TodoList) *TodoItem {
	ti := new(TodoItem)
	ti.Title = title
	ti.Content = content
	ti.Parent = parent
	return ti
}

func NewTodoList() *TodoList {
	tl := new(TodoList)

	// Bind events in go rather than Render() template
	tl.Events(cas9.SelectorEvent{
		".css .selector": cas9.Event{"click", func(e cas9.EventObj) {

		}},
	})

	tl.Items = TodoItems{
		NewTodoItem("testnew", "test", tl),
		NewTodoItem("testnew2", "testnew2", tl),
	}

	return tl
}

func main() {
	cas9.StartApp("#app", NewTodoList())
}
```

### Build and Run

$ `GOOS=js GOARCH=wasm go build -o main.wasm myapp && goexec 'http.ListenAndServe(":8080", http.FileServer(http.Dir(".")))'`

### Enjoy
