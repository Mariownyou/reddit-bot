package bot

import "testing"

func TestParsePost(t *testing.T) {
	m := Manager{}
	m.Data = NewContext()

	m.ParsePost("Title: Hello world\nhello: test, message\nempty: , success")
	if m.Data.flairs["hello"] != "test" {
		t.Error("flair not found for sub hello", m.Data.flairs["hello"])
	}

	if m.Data.flairs["empty"] != "" {
		t.Error("flair not found for sub empty", m.Data.flairs["empty"])
	}
}
