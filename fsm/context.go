package fsm

type Context map[string]interface{}

func (c Context) Set(key string, value interface{}) {
	c[key] = value
}

func (c Context) Get(key string) interface{} {
	return c[key]
}
