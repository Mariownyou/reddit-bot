package bot

type Context struct {
	flairs    map[string]string
	subs      []string
	caption   string
	file      []byte
	preview   []byte
	filetype  string
	albumSubs []string
	tweet     bool
}

func NewContext() Context {
	return Context{
		flairs: map[string]string{},
		subs:   []string{},
	}
}
