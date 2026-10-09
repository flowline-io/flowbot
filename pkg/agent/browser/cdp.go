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
	"github.com/chromedp/chromedp/remote"
)

type chromedpPage struct {
	allocCancel context.CancelFunc
	ctx         context.Context
	cancel      context.CancelFunc
}

func openCDPPage(ctx context.Context, cfg Config) (cdpPage, error) {
	allocCtx, allocCancel := remote.NewAllocator(ctx, cfg.Endpoint, remote.NoModifyURL)
	tabCtx, cancel := chromedp.NewContext(allocCtx, chromedp.WithNewBrowserContext())
	if err := chromedp.Do(tabCtx); err != nil {
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

func (p *chromedpPage) withOp(opCtx context.Context) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(p.ctx)
	stop := context.AfterFunc(opCtx, cancel)
	return ctx, func() {
		stop()
		cancel()
	}
}

func (p *chromedpPage) run(opCtx context.Context, actions ...chromedp.Action[chromedp.Void]) error {
	ctx, cancel := p.withOp(opCtx)
	defer cancel()
	return chromedp.Do(ctx, actions...)
}

func runVal[T any](opCtx context.Context, p *chromedpPage, action chromedp.Action[T]) (T, error) {
	ctx, cancel := p.withOp(opCtx)
	defer cancel()
	return chromedp.Run(ctx, action)
}

func (p *chromedpPage) Navigate(ctx context.Context, rawURL string) (string, error) {
	if err := p.run(ctx, chromedp.Navigate(rawURL)); err != nil {
		return "", fmt.Errorf("navigate: %w", err)
	}
	final, err := runVal(ctx, p, chromedp.Location())
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
	raw, err := runVal(ctx, p, chromedp.Evaluate[string](snapshotDOMJS))
	if err != nil {
		return "", nil, fmt.Errorf("snapshot: %w", err)
	}
	return decodeSnapshot(raw)
}

func (p *chromedpPage) snapshotAX(ctx context.Context) (string, map[string]struct{}, error) {
	var tree string
	var refs map[string]struct{}
	err := p.run(ctx, chromedp.Func(func(ctx context.Context, t *chromedp.Target) error {
		if err := clearFlowbotRefs(ctx, t); err != nil {
			return err
		}
		res, err := cdp.Call(ctx, t, accessibility.GetFullAXTree, accessibility.GetFullAXTreeParams{})
		if err != nil {
			return err
		}
		lines := make([]string, 0, len(res.Nodes))
		refs = make(map[string]struct{})
		refN := 0
		for _, n := range res.Nodes {
			if n == nil || !axInteractive(n) {
				continue
			}
			ref := fmt.Sprintf("e%d", refN)
			refN++
			if n.BackendDOMNodeID != 0 {
				if err := stampRefOnBackendNode(ctx, t, n.BackendDOMNodeID, ref); err != nil {
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
		title, err := chromedp.Title()(ctx, t)
		if err != nil {
			return err
		}
		loc, err := chromedp.Location()(ctx, t)
		if err != nil {
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

func clearFlowbotRefs(ctx context.Context, t *chromedp.Target) error {
	_, err := chromedp.Evaluate[bool](`(function(){
  document.querySelectorAll('[`+refAttr+`]').forEach(el => el.removeAttribute('`+refAttr+`'));
  return true;
})()`)(ctx, t)
	return err
}

func stampRefOnBackendNode(ctx context.Context, t *chromedp.Target, backendID cdp.BackendNodeID, ref string) error {
	resolved, err := cdp.Call(ctx, t, dom.ResolveNode, dom.ResolveNodeParams{BackendNodeID: backendID})
	if err != nil || resolved.Object == nil || resolved.Object.ObjectID == "" {
		return fmt.Errorf("resolve node: %w", err)
	}
	_, err = cdp.Call(ctx, t, runtime.CallFunctionOn, runtime.CallFunctionOnParams{
		FunctionDeclaration: `function(attr, ref){ this.setAttribute(attr, ref); }`,
		ObjectID:            resolved.Object.ObjectID,
		Arguments: []*runtime.CallArgument{
			{Value: mustJSON(refAttr)},
			{Value: mustJSON(ref)},
		},
	})
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
	sel := chromedp.CSS(refSelector(ref))
	err := p.run(ctx, chromedp.Click(sel))
	if err != nil {
		return fmt.Errorf("click %s: %w", ref, err)
	}
	return nil
}

func (p *chromedpPage) Type(ctx context.Context, ref, text string) error {
	sel := chromedp.CSS(refSelector(ref))
	err := p.run(ctx,
		chromedp.Focus(sel),
		chromedp.SendKeys(sel, text),
	)
	if err != nil {
		return fmt.Errorf("type %s: %w", ref, err)
	}
	return nil
}

func (p *chromedpPage) Scroll(ctx context.Context, dx, dy int) error {
	err := p.run(ctx, chromedp.Evaluate[chromedp.Void](fmt.Sprintf("window.scrollBy(%d,%d)", dx, dy)))
	if err != nil {
		return fmt.Errorf("scroll: %w", err)
	}
	return nil
}

func (p *chromedpPage) Wait(ctx context.Context, ms int, quirks Quirks) error {
	actions := make([]chromedp.Action[chromedp.Void], 0, 2)
	if quirks.PreferNetworkIdle {
		actions = append(actions, chromedp.WaitReady(chromedp.CSS("body")))
	}
	actions = append(actions, chromedp.Sleep(time.Duration(ms)*time.Millisecond))
	if err := p.run(ctx, actions...); err != nil {
		return fmt.Errorf("wait: %w", err)
	}
	return nil
}

func (p *chromedpPage) Screenshot(ctx context.Context) ([]byte, error) {
	buf, err := runVal(ctx, p, chromedp.FullScreenshot(90))
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
