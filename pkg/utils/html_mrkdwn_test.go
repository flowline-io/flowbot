package utils

import "testing"

func TestHTMLToMrkdwn(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "empty", in: "", want: ""},
		{name: "bold", in: "<b>bold</b>", want: "*bold*"},
		{name: "italic and link", in: `<em>hi</em> <a href="https://ex.com">Ex</a>`, want: "_hi_ <https://ex.com|Ex>"},
		{name: "autolink same text", in: `<a href="https://ex.com">https://ex.com</a>`, want: "<https://ex.com>"},
		{name: "paragraphs", in: "<p>one</p><p>two</p>", want: "one\ntwo"},
		{name: "strip unknown", in: "<script>x</script><span>ok</span>", want: "ok"},
		{name: "code", in: "<code>cmd</code>", want: "`cmd`"},
		{name: "strike", in: "<del>gone</del>", want: "~gone~"},
		{name: "heading", in: "<h1 id=\"title\">Title</h1>", want: "*Title*"},
		{name: "blockquote", in: "<blockquote><p>quote</p></blockquote>", want: "> quote"},
		{name: "hr", in: "<p>a</p><hr><p>b</p>", want: "a\n---\nb"},
		{name: "hr spaced", in: "<p>a</p>\n<hr>\n<p>b</p>", want: "a\n\n---\n\nb"},
		{name: "ul", in: "<ul><li>a</li><li>b</li></ul>", want: "• a\n• b"},
		{name: "ol", in: "<ol><li>a</li><li>b</li></ol>", want: "1. a\n2. b"},
		{
			name: "nested ul",
			in:   "<ul><li>a</li><li>b<ul><li>nested<ul><li>deep</li></ul></li></ul></li><li>c</li></ul>",
			want: "• a\n• b\n    ◦ nested\n        ▪ deep\n• c",
		},
		{
			name: "nested blockquote",
			in:   "<blockquote><p>outer</p><blockquote><p>inner</p></blockquote></blockquote>",
			want: "> outer\n> > inner",
		},
		{name: "fragment link becomes text", in: `<a href="#title">top</a>`, want: "top"},
		{name: "image", in: `<img src="https://ex.com/i.png" alt="alt">`, want: "<https://ex.com/i.png|alt>"},
		{name: "table row", in: "<tr><th>A</th><th>B</th></tr><tr><td>1</td><td>2</td></tr>", want: "A | B\n1 | 2"},
		{
			name: "pre code block",
			in:   "<pre><code>name: url_report\npipeline: [check_url]\n</code></pre>",
			want: "```\nname: url_report\npipeline: [check_url]\n```",
		},
		{
			name: "pre without code",
			in:   "<pre>plain\nblock</pre>",
			want: "```\nplain\nblock\n```",
		},
		{
			name: "inline code beside pre",
			in:   "<p>run <code>flowbot workflow get</code></p><pre><code>ok</code></pre>",
			want: "run `flowbot workflow get`\n```\nok\n```",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HTMLToMrkdwn(tt.in)
			if got != tt.want {
				t.Fatalf("HTMLToMrkdwn(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestMarkdownToMrkdwn(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "empty", in: "", want: ""},
		{name: "bold", in: "**bold**", want: "*bold*"},
		{name: "italic", in: "*italic*", want: "_italic_"},
		{name: "strikethrough", in: "~~gone~~", want: "~gone~"},
		{name: "link", in: "[Ex](https://ex.com)", want: "<https://ex.com|Ex>"},
		{name: "autolink", in: "see https://ex.com please", want: "see <https://ex.com> please"},
		{name: "heading", in: "# Title", want: "*Title*"},
		{name: "blockquote", in: "> quote", want: "> quote"},
		{name: "ul", in: "- a\n- b", want: "• a\n• b"},
		{name: "ol", in: "1. a\n2. b", want: "1. a\n2. b"},
		{
			name: "nested ul",
			in:   "- a\n- b\n  - nested\n    - deep\n- c",
			want: "• a\n• b\n    ◦ nested\n        ▪ deep\n• c",
		},
		{
			name: "nested ol",
			in:   "1. a\n2. b\n   1. nested\n3. c",
			want: "1. a\n2. b\n    1. nested\n3. c",
		},
		{name: "hr", in: "a\n\n---\n\nb", want: "a\n\n---\n\nb"},
		{name: "table", in: "| A | B |\n| - | - |\n| 1 | 2 |", want: "A | B\n\n1 | 2"},
		{name: "image", in: "![alt](https://ex.com/i.png)", want: "<https://ex.com/i.png|alt>"},
		{name: "task list", in: "- [x] done\n- [ ] todo", want: "• ✅ done\n• ⬜ todo"},
		{name: "fragment link", in: "[top](#title)", want: "top"},
		{name: "list with bold", in: "- **读写文件**: 查看代码", want: "• *读写文件*: 查看代码"},
		{name: "plain text", in: "hello", want: "hello"},
		{
			name: "fenced code block",
			in:   "```yaml\nname: url_report\npipeline: [check_url]\n```\n",
			want: "```\nname: url_report\npipeline: [check_url]\n```",
		},
		{
			name: "inline code unchanged",
			in:   "use `inputs` and `mapper`",
			want: "use `inputs` and `mapper`",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MarkdownToMrkdwn(tt.in)
			if got != tt.want {
				t.Fatalf("MarkdownToMrkdwn(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
