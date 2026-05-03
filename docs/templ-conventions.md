# templ Conventions

## Basic Syntax

- templ components compile to Go functions returning `templ.Component`
- Use `{ variable }` for escaped output, `{ templ.Raw(var) }` for unescaped HTML
- Use `templ.KV("class", condition)` for conditional CSS classes
- Use `templ.SafeURL()` for `href` and `src` attributes to prevent JavaScript URL injection
- Raw string interpolation `{ var }` inside raw HTML does NOT escape — be careful

## Security

### XSS Prevention

**Never** build HTML via `fmt.Sprintf` with unescaped user/i18n data. This is a stored XSS vulnerability.

```go
// WRONG - XSS vulnerability
func inlineLink(href, text string) string {
    return fmt.Sprintf(`<a href="%s">%s</a>`, href, text)
}

// CORRECT - escape user data
func inlineLink(href, text string) string {
    return fmt.Sprintf(`<a href="%s" class="...">%s</a>`,
        href,
        templ.EscapeString(text),
    )
}
```

### Attribute Safety

- **href/src**: Always wrap dynamic URLs in `templ.SafeURL()`
- **class**: Use `templ.KV()` for conditionals; `templ.SafeClass()` bypasses sanitization (use carefully)
- **style**: Cannot be expressions; use `style=` with constant strings only
- **text content**: `{ var }` is auto-escaped. Use `{ templ.Raw(var) }` only for trusted HTML

### Code-Only Components

When implementing `templ.Component` in Go, always use `templ.EscapeString()` for user data:

```go
func button(text string) templ.Component {
    return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
        _, err := io.WriteString(w, "<button>"+templ.EscapeString(text)+"</button>")
        return err
    })
}
```

## View Models

Create view model types that map directly to your UI structure. Handlers build the view model, templates consume it. This makes rendering testable without integration tests.

```go
type InviteComponentViewModel struct {
    InviteCount  int
    ErrorMessage string
}

templ teamInviteComponent(model InviteComponentViewModel) {
    if model.InviteCount > 0 {
        <div>You have { fmt.Sprintf("%d", model.InviteCount) } pending invites</div>
    }
    if model.ErrorMessage != "" {
        <div class="error">{ model.ErrorMessage }</div>
    }
}
```

## Testing

Use `goquery` to test rendered HTML. Pipe component output through `io.Pipe` into `goquery.NewDocumentFromReader`.

```go
func TestHeader(t *testing.T) {
    r, w := io.Pipe()
    go func() {
        _ = headerTemplate("Posts").Render(context.Background(), w)
        _ = w.Close()
    }()
    doc, err := goquery.NewDocumentFromReader(r)
    if err != nil {
        t.Fatalf("failed to read template: %v", err)
    }
    if doc.Find(`[data-testid="headerTemplate"]`).Length() == 0 {
        t.Error("expected data-testid attribute to be rendered")
    }
}
```

- Add `data-testid` attributes to components to make CSS selectors less brittle
- Test components in isolation; HTTP handlers check that templates render, not specific field values

## HTMX Conventions

- Use `hx-get`, `hx-post`, `hx-delete`, `hx-swap`, `hx-target` for AJAX interactions
- Loading states: use `htmx-idle` / `htmx-busy` span pattern for button loading indicators
- Use `hx-disabled-elt="this"` to disable the triggering element during requests
- HTMX attributes should target elements by `id` (e.g., `hx-target="#leaderboard"`)

## Content Security Policy

Use `templ.WithNonce(ctx, nonce)` to set nonces for inline scripts. The nonce is applied to all `<script>` tags rendered by templ.

## IDE Support

- **VS Code**: Install the [templ](https://marketplace.visualstudio.com/items?itemName=a-h.templ) extension
  - Enable format on save, Tailwind CSS intellisense, and Emmet for `.templ` files in settings.json
- **Neovim**: Install `tree-sitter-templ` for syntax highlighting; configure LSP with `lspconfig.templ.setup`
- **JetBrains**: Install the [templ](https://plugins.jetbrains.com/plugin/23088-templ) plugin
