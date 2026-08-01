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
		{name: "paragraphs", in: "<p>one</p><p>two</p>", want: "one\ntwo"},
		{name: "strip unknown", in: "<script>x</script><span>ok</span>", want: "ok"},
		{name: "code", in: "<code>cmd</code>", want: "`cmd`"},
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
