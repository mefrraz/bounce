# Test Fixtures

Place real FPB HTML files here for offline testing.
These snapshots let us develop and test the scraper even
when the basketball season is over.

## Files needed:

- `fpb-calendario-{competicao}.html` — competition schedule page
- `fpb-resultados-{competicao}.html` — competition results page
- `fpb-classificacao-{competicao}.html` — standings page
- `fpb-ficha-jogo-{id}.html` — game detail page
- `fpb-clube-{id}.html` — club teams page
- `fpb-equipa-{id}.html` — team detail page
- `fpb-atleta-{id}.html` — athlete page
- `fpb-ajax-*.html` — WordPress AJAX responses

Download real pages with:
    curl -H "User-Agent: Bounce/0.1" "https://www.fpb.pt/calendario/12345" > testdata/fpb-calendario-12345.html
