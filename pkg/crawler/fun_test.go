package crawler

import (
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseFun(t *testing.T) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(`<tr class="athing" id="25782861">
      <td align="right" valign="top" class="title"><span class="rank">3.</span></td>      
		<td valign="top" class="votelinks"><center><a id="up_25782861" href="vote?id=25782861&amp;how=up&amp;goto=news">
		<div class="votearrow" title="upvote"></div></a></center></td><td class="title">
		<a href="http://demo.com" class="storylink">demo</a>
		<span class="sitebit comhead"> (<a href="from?site=demo.com">
		<span class="sitestr">demo.com</span></a>)
		</span></td></tr>`))
	require.NoError(t, err)

	sel := doc.First()
	t.Parallel()
	tests := []struct {
		name   string
		fun    string
		expect string
	}{
		{
			name:   "text selector",
			fun:    `$("a.storylink").text`,
			expect: "demo",
		},
		{
			name:   "expand regex capture",
			fun:    `$(".rank").text.expand("(\d+)", "#$1")`,
			expect: "#3",
		},
		{
			name:   "match regex capture",
			fun:    `$(".rank").text.match("(\d+)")`,
			expect: "3",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := ParseFun(sel, tt.fun)
			r, err := f.Invoke()
			require.NoError(t, err)
			assert.Equal(t, tt.expect, r)
		})
	}
}
