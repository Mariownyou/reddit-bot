package fsm

type Context map[string]interface{}

func NewContext() Context {
	return Context{
		"flairs":  map[string]string{},
		"subs":    []string{},
		"caption": "",
		"link":    "",
	}
}

func (c Context) Set(key string, value interface{}) {
	c[key] = value
}

func (c Context) Get(key string) interface{} {
	return c[key]
}
