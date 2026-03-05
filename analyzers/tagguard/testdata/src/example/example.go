package example

type Good struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

type Bad struct {
	UserName string `json:"userName"` // want `\[tagguard\] json tag "userName" on field UserName should be snake_case`
	Missing  string // want `\[tagguard\] exported field Missing has no json tag`
	Omitted  string `json:"-"` // OK — explicitly omitted
	private  string // OK — unexported
}
