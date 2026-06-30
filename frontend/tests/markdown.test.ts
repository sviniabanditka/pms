import { describe, it, expect } from 'vitest'
import { renderMarkdown } from '../composables/useMarkdown'

describe('renderMarkdown — escaping / XSS', () => {
  it('empty input → empty string', () => {
    expect(renderMarkdown('')).toBe('')
  })

  it('escapes angle brackets (no raw HTML)', () => {
    const out = renderMarkdown('<script>alert(1)</script>')
    expect(out).not.toContain('<script>')
    expect(out).toContain('&lt;script&gt;')
  })

  it('escapes quotes so attribute injection in image alt cannot break out', () => {
    const out = renderMarkdown('![" onerror="alert(1)](x.png)')
    // the quote is escaped before the image rule runs → no real onerror attribute
    expect(out).not.toContain('onerror="')
    expect(out).toContain('&quot;')
    expect(out).toContain('src="x.png"')
  })
})

describe('renderMarkdown — inline formatting', () => {
  it('bold', () => {
    expect(renderMarkdown('a **b** c')).toContain('<strong>b</strong>')
  })
  it('italic with asterisks and underscores', () => {
    expect(renderMarkdown('an *em* word')).toContain('<em>em</em>')
    expect(renderMarkdown('an _em_ word')).toContain('<em>em</em>')
  })
  it('inline code is not formatted inside', () => {
    const out = renderMarkdown('`a_b*c`')
    expect(out).toContain('<code')
    expect(out).not.toContain('<em>')
    expect(out).not.toContain('<strong>')
  })
})

describe('renderMarkdown — the underscore-in-URL gotcha', () => {
  it('does not turn underscores inside an image URL into <em>', () => {
    const out = renderMarkdown('![shot](http://x/Screenshot_2026_06_30_at.png)')
    expect(out).toContain('src="http://x/Screenshot_2026_06_30_at.png"')
    expect(out).not.toContain('<em>')
  })
})

describe('renderMarkdown — links / mentions / chips', () => {
  it('markdown link', () => {
    const out = renderMarkdown('[site](https://example.com)')
    expect(out).toContain('href="https://example.com"')
    expect(out).toContain('>site</a>')
  })
  it('mention @[Name](uuid)', () => {
    const out = renderMarkdown('hi @[Maya](11111111-1111-1111-1111-111111111111)')
    expect(out).toContain('@Maya')
  })
  it('shorthand #KEY-N becomes an issue ref chip', () => {
    const out = renderMarkdown('see #ENG-42 today')
    expect(out).toContain('data-issue="ENG-42"')
  })
  it('image', () => {
    const out = renderMarkdown('![a](u.png)')
    expect(out).toContain('<img src="u.png"')
    expect(out).toContain('alt="a"')
  })
})

describe('renderMarkdown — block structure', () => {
  it('headings', () => {
    expect(renderMarkdown('# H1')).toContain('<h1')
    expect(renderMarkdown('## H2')).toContain('<h2')
    expect(renderMarkdown('### H3')).toContain('<h3')
  })
  it('bullet list groups consecutive items in one <ul>', () => {
    const out = renderMarkdown('- a\n- b')
    expect(out).toContain('<ul')
    expect((out.match(/<li>/g) || []).length).toBe(2)
    expect((out.match(/<ul/g) || []).length).toBe(1)
  })
  it('fenced code block', () => {
    const out = renderMarkdown('```\nx := 1\n```')
    expect(out).toContain('<pre')
    expect(out).toContain('<code>')
    expect(out).toContain('x := 1')
  })
  it('plain line becomes a paragraph', () => {
    expect(renderMarkdown('hello')).toBe('<p>hello</p>')
  })
})
