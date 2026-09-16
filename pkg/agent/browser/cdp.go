package browser

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/chromedp/cdproto/accessibility"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/dom"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
)

type chromedpPage struct {
	allocCancel context.CancelFunc
	ctx         context.Context
	cancel      context.CancelFunc
}

func openCDPPage(ctx context.Context, cfg Config) (cdpPage, error) {
	allocCtx, allocCancel := chromedp.NewRemoteAllocator(ctx, cfg.Endpoint, chromedp.NoModifyURL)
	tabCtx, cancel := chromedp.NewContext(allocCtx, chromedp.WithNewBrowserContext())
	if err := chromedp.Run(tabCtx); err != nil {
		cancel()
		allocCancel()
		return nil, fmt.Errorf("browser: connect %s: %w", cfg.Endpoint, err)
	}
	return &chromedpPage{
		allocCancel: allocCancel,
		ctx:         tabCtx,
		cancel:      cancel,
	}, nil
}

func (p *chromedpPage) Close() error {
	if p.cancel != nil {
		p.cancel()
	}
	if p.allocCancel != nil {
		p.allocCancel()
	}
	return nil
}

func (p *chromedpPage) run(opCtx context.Context, actions ...chromedp.Action) error {
	ctx, cancel := context.WithCancel(p.ctx)
	defer cancel()
	stop := context.AfterFunc(opCtx, cancel)
	defer stop()
	return chromedp.Run(ctx, actions...)
}

func (p *chromedpPage) Navigate(ctx context.Context, rawURL string) (string, error) {
	var final string
	err := p.run(ctx,
		chromedp.Navigate(rawURL),
		chromedp.Location(&final),
	)
	if err != nil {
		return "", fmt.Errorf("navigate: %w", err)
	}
	return final, nil
}

func (p *chromedpPage) Snapshot(ctx context.Context) (string, map[string]struct{}, error) {
	tree, refs, err := p.snapshotAX(ctx)
	if err == nil && len(refs) > 0 {
		return tree, refs, nil
	}
	return p.snapshotDOM(ctx)
}

func (p *chromedpPage) snapshotDOM(ctx context.Context) (string, map[string]struct{}, error) {
	var raw string
	err := p.run(ctx, chromedp.Evaluate(snapshotDOMJS, &raw))
	if err != nil {
		return "", nil, fmt.Errorf("snapshot: %w", err)
	}
	return decodeSnapshot(raw)
}

func (p *chromedpPage) snapshotAX(ctx context.Context) (string, map[string]struct{}, error) {
	var tree string
	var refs map[string]struct{}
	err := p.run(ctx, chromedp.ActionFunc(func(ctx context.Context) error {
		if err := clearFlowbotRefs(ctx); err != nil {
			return err
		}
		nodes, err := accessibility.GetFullAXTree().Do(ctx)
		if err != nil {
			return err
		}
		lines := make([]string, 0, len(nodes))
		refs = make(map[string]struct{})
		refN := 0
		for _, n := range nodes {
			if n == nil || !axInteractive(n) {
				continue
			}
			ref := fmt.Sprintf("e%d", refN)
			refN++
			if n.BackendDOMNodeID != 0 {
				if err := stampRefOnBackendNode(ctx, n.BackendDOMNodeID, ref); err != nil {
					continue
				}
			}
			refs[ref] = struct{}{}
			role := axRole(n)
			name := axName(n)
			line := "- " + role
			if name != "" {
				line += ` "` + strings.ReplaceAll(name, `"`, `\"`) + `"`
			}
			line += " [ref=" + ref + "]"
			lines = append(lines, line)
		}
		var title, loc string
		if err := chromedp.Title(&title).Do(ctx); err != nil {
			return err
		}
		if err := chromedp.Location(&loc).Do(ctx); err != nil {
			return err
		}
		body := "(no interactive nodes)"
		if len(lines) > 0 {
			body = strings.Join(lines, "\n")
		}
		tree = "url: " + loc + "\ntitle: " + title + "\n" + body
		return nil
	}))
	if err != nil {
		return "", nil, err
	}
	return tree, refs, nil
}

func clearFlowbotRefs(ctx context.Context) error {
	var ignored bool
	return chromedp.Evaluate(`(function(){
  document.querySelectorAll('[`+refAttr+`]').forEach(el => el.removeAttribute('`+refAttr+`'));
  return true;
})()`, &ignored).Do(ctx)
}

func stampRefOnBackendNode(ctx context.Context, backendID cdp.BackendNodeID, ref string) error {
	remote, err := dom.ResolveNode().WithBackendNodeID(backendID).Do(ctx)
	if err != nil || remote == nil || remote.ObjectID == "" {
		return fmt.Errorf("resolve node: %w", err)
	}
	_, _, err = runtime.CallFunctionOn(`function(attr, ref){ this.setAttribute(attr, ref); }`).
		WithObjectID(remote.ObjectID).
		WithArguments([]*runtime.CallArgument{
			{Value: mustJSON(refAttr)},
			{Value: mustJSON(ref)},
		}).Do(ctx)
	return err
}

func mustJSON(v string) []byte {
	b, err := sonic.Marshal(v)
	if err != nil {
		return []byte(`""`)
	}
	return b
}

func axInteractive(n *accessibility.Node) bool {
	if n == nil || n.Ignored {
		return false
	}
	role := strings.ToLower(axRole(n))
	switch role {
	case "button", "link", "textbox", "searchbox", "combobox", "checkbox",
		"radio", "switch", "tab", "menuitem", "option", "slider", "spinbutton",
		"listbox", "treeitem":
		return true
	default:
		return false
	}
}

func axRole(n *accessibility.Node) string {
	if n == nil {
		return ""
	}
	return axString(n.Role)
}

func axName(n *accessibility.Node) string {
	if n == nil {
		return ""
	}
	name := axString(n.Name)
	if len(name) > 80 {
		return name[:80]
	}
	return name
}

func axString(v *accessibility.Value) string {
	if v == nil || len(v.Value) == 0 {
		return ""
	}
	raw := strings.TrimSpace(string(v.Value))
	var s string
	if err := sonic.UnmarshalString(raw, &s); err == nil {
		return strings.TrimSpace(s)
	}
	return strings.Trim(raw, `"`)
}

func decodeSnapshot(raw string) (string, map[string]struct{}, error) {
	var payload snapshotPayload
	if err := sonic.UnmarshalString(raw, &payload); err != nil {
		return "", nil, fmt.Errorf("snapshot decode: %w", err)
	}
	refs := make(map[string]struct{}, len(payload.Refs))
	for _, ref := range payload.Refs {
		refs[ref] = struct{}{}
	}
	return payload.Tree, refs, nil
}

func (p *chromedpPage) Click(ctx context.Context, ref string) error {
	sel := refSelector(ref)
	err := p.run(ctx, chromedp.Click(sel, chromedp.ByQuery))
	if err != nil {
		return fmt.Errorf("click %s: %w", ref, err)
	}
	return nil
}

func (p *chromedpPage) Type(ctx context.Context, ref, text string) error {
	sel := refSelector(ref)
	err := p.run(ctx,
		chromedp.Focus(sel, chromedp.ByQuery),
		chromedp.SendKeys(sel, text, chromedp.ByQuery),
	)
	if err != nil {
		return fmt.Errorf("type %s: %w", ref, err)
	}
	return nil
}

func (p *chromedpPage) Scroll(ctx context.Context, dx, dy int) error {
	err := p.run(ctx, chromedp.Evaluate(fmt.Sprintf("window.scrollBy(%d,%d)", dx, dy), nil))
	if err != nil {
		return fmt.Errorf("scroll: %w", err)
	}
	return nil
}

func (p *chromedpPage) Wait(ctx context.Context, ms int, quirks Quirks) error {
	actions := make([]chromedp.Action, 0, 2)
	if quirks.PreferNetworkIdle {
		actions = append(actions, chromedp.WaitReady("body", chromedp.ByQuery))
	}
	actions = append(actions, chromedp.Sleep(time.Duration(ms)*time.Millisecond))
	if err := p.run(ctx, actions...); err != nil {
		return fmt.Errorf("wait: %w", err)
	}
	return nil
}

func (p *chromedpPage) Screenshot(ctx context.Context) ([]byte, error) {
	var buf []byte
	err := p.run(ctx, chromedp.FullScreenshot(&buf, 90))
	if err != nil {
		return nil, fmt.Errorf("screenshot: %w", err)
	}
	return buf, nil
}

func refSelector(ref string) string {
	return fmt.Sprintf(`[%s="%s"]`, refAttr, ref)
}

type snapshotPayload struct {
	Tree string   `json:"tree"`
	Refs []string `json:"refs"`
}

// snapshotDOMJS builds a compact interactive DOM-role tree and stamps refs on nodes.
const snapshotDOMJS = `(function(){
  const attr = '` + refAttr + `';
  document.querySelectorAll('['+attr+']').forEach(el => el.removeAttribute(attr));
  const refs = [];
  const lines = [];
  const interesting = new Set(['A','BUTTON','INPUT','SELECT','TEXTAREA','SUMMARY','OPTION']);
  function roleOf(el){
    const explicit = (el.getAttribute('role')||'').trim();
    if (explicit) return explicit;
    const tag = el.tagName;
    if (tag === 'A') return 'link';
    if (tag === 'BUTTON') return 'button';
    if (tag === 'INPUT') return (el.getAttribute('type')||'text').toLowerCase();
    if (tag === 'SELECT') return 'combobox';
    if (tag === 'TEXTAREA') return 'textbox';
    if (tag === 'SUMMARY') return 'button';
    return tag.toLowerCase();
  }
  function labelOf(el){
    const aria = (el.getAttribute('aria-label')||'').trim();
    if (aria) return aria.slice(0,80);
    if (el.tagName === 'INPUT' || el.tagName === 'TEXTAREA') {
      return ((el.getAttribute('placeholder')||el.value||'')+'').slice(0,80);
    }
    const t = (el.innerText||el.textContent||'').replace(/\s+/g,' ').trim();
    return t.slice(0,80);
  }
  function walk(el, depth){
    if (!el || el.nodeType !== 1 || depth > 12) return;
    const style = window.getComputedStyle(el);
    if (style && (style.display === 'none' || style.visibility === 'hidden')) return;
    const tag = el.tagName;
    const interactive = interesting.has(tag) || el.hasAttribute('onclick') || el.getAttribute('tabindex') !== null || (el.getAttribute('role')||'') !== '';
    let nextDepth = depth;
    if (interactive) {
      const ref = 'e' + refs.length;
      refs.push(ref);
      el.setAttribute(attr, ref);
      const name = labelOf(el);
      lines.push('  '.repeat(depth) + '- ' + roleOf(el) + (name ? ' "'+name.replace(/"/g,'\\"')+'"' : '') + ' [ref='+ref+']');
      nextDepth = depth + 1;
    }
    for (const child of el.children) walk(child, nextDepth);
  }
  walk(document.body, 0);
  const title = document.title || '';
  const url = location.href || '';
  const header = 'url: '+url+'\ntitle: '+title+'\n';
  return JSON.stringify({tree: header + (lines.length ? lines.join('\n') : '(no interactive nodes)'), refs: refs});
})()`
