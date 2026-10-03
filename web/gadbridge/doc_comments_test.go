package gadbridge

import "testing"

// The doc comments of a file, as the IDE's Docs panel asks them: the file's
// own (/*** … ***/) is "root"; a block (/** … **/) and lines (///) are
// "block" and "single", titled by the line of code under them; a plain
// comment is none.
func TestDocCommentsKinds(t *testing.T) {
	src := "/***\n# The module\n\nWhat it is.\n***/\n\n" +
		"// a plain comment\n\n" +
		"/** What a page tells its layout. **/\nexport class Default {\n}\n\n" +
		"/// The list.\n/// Of posts.\nexport class PostList {\n}\n"
	docs := DocComments(src)
	want := []DocComment{
		{Kind: "root", Content: "# The module\n\nWhat it is."},
		{Kind: "block", Title: "export class Default {", Content: " What a page tells its layout. "},
		{Kind: "single", Title: "export class PostList {", Content: "The list.\nOf posts."},
	}
	if len(docs) != len(want) {
		t.Fatalf("%d docs, want %d: %+v", len(docs), len(want), docs)
	}
	for i, w := range want {
		d := docs[i]
		if d.Kind != w.Kind || d.Content != w.Content || (w.Title != "" && d.Title != w.Title) {
			t.Errorf("doc %d: %+v, want %+v", i, d, w)
		}
	}
}
