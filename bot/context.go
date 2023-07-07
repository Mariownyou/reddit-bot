package bot

type Context struct {
	flairs      map[string]string
	subs        []string
	caption     string
	link        string
	previewLink string
	file        []byte
	preview     []byte
}

func NewContext() Context {
	return Context{
		flairs: map[string]string{},
		subs:   []string{},
	}
}
