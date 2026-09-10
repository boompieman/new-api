import assert from 'node:assert/strict'
import test from 'node:test'

import worker from '../../public/_worker.js'

for (const path of ['/static/js/async/missing.js', '/static/css/missing.css']) {
  test(path + ' does not cache the SPA fallback as an asset', async () => {
    const response = await worker.fetch(
      new Request('https://example.com' + path),
      {
        ASSETS: {
          fetch: async () =>
            new Response('<html>app</html>', {
              headers: {
                'Content-Type': 'text/html',
                'Cache-Control': 'public, max-age=31536000, immutable',
              },
            }),
        },
      }
    )
    assert.equal(response.status, 404)
    assert.equal(response.headers.get('Cache-Control'), 'no-store')
    assert.equal(await response.text(), 'Not found')
  })
}

test('valid immutable assets retain their response and caching', async () => {
  const asset = new Response('body {}', {
    headers: {
      'Content-Type': 'text/css',
      'Cache-Control': 'public, max-age=31536000, immutable',
    },
  })
  assert.equal(
    await worker.fetch(new Request('https://example.com/static/css/app.css'), {
      ASSETS: { fetch: async () => asset },
    }),
    asset
  )
})

test('dashboard routes still receive the SPA without immutable caching', async () => {
  const response = await worker.fetch(
    new Request('https://example.com/dashboard/overview'),
    {
      ASSETS: {
        fetch: async () =>
          new Response('<html>app</html>', {
            headers: { 'Content-Type': 'text/html' },
          }),
      },
    }
  )
  assert.equal(response.status, 200)
  assert.equal(response.headers.get('Cache-Control'), 'no-cache')
  assert.equal(await response.text(), '<html>app</html>')
})
