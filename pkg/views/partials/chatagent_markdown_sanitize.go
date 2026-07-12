package partials

import (
	"regexp"
	"sync"

	"github.com/microcosm-cc/bluemonday"
)

var (
	chatAgentMarkdownPolicyOnce sync.Once
	chatAgentMarkdownPolicy     *bluemonday.Policy
)

// chatAgentMarkdownSanitizer returns a policy that extends UGC rules with KaTeX
// MathML and layout attributes produced by goldmark-katex.
func chatAgentMarkdownSanitizer() *bluemonday.Policy {
	chatAgentMarkdownPolicyOnce.Do(func() {
		p := bluemonday.UGCPolicy()
		p.AllowElements(
			"math", "semantics", "mrow", "msup", "msub", "msubsup", "mfrac", "msqrt", "mroot",
			"mi", "mn", "mo", "mtext", "mspace", "mstyle", "annotation", "mpadded", "mphantom",
			"menclose", "mover", "munder", "munderover", "mtable", "mtr", "mtd", "maligngroup",
			"malignmark", "mlabeledtr", "merror", "mprescripts", "none", "mrow",
		)
		p.AllowAttrs("xmlns").Matching(regexp.MustCompile(`^http://www\.w3\.org/1998/Math/MathML$`)).OnElements("math")
		p.AllowAttrs("encoding").Matching(regexp.MustCompile(`^application/x-tex$`)).OnElements("annotation")
		p.AllowAttrs("class").Matching(regexp.MustCompile(`^[a-zA-Z0-9_\- ]+$`)).OnElements("span")
		p.AllowAttrs("style").Matching(regexp.MustCompile(`^[-a-zA-Z0-9:;.%(), empx]+$`)).OnElements("span")
		p.AllowAttrs("aria-hidden").Matching(regexp.MustCompile(`^(?:true|false)$`)).OnElements("span")
		chatAgentMarkdownPolicy = p
	})
	return chatAgentMarkdownPolicy
}
