package bot

type Context struct {
	flairs   map[string]string
	subs     []string
	caption  string
	file     []byte
	filetype string
}

func NewContext() Context {
	return Context{
		flairs: map[string]string{},
		subs:   []string{},
	}
}
