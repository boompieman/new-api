const BACKEND_PREFIXES = [
  '/api',
  '/v1',
  '/v1beta',
  '/pg',
  '/mj',
  '/suno',
  '/kling',
  '/jimeng',
  '/dashboard/billing',
]

function hasPathPrefix(pathname, prefix) {
  return pathname === prefix || pathname.startsWith(`${prefix}/`)
}

function shouldProxyToBackend(pathname) {
  if (BACKEND_PREFIXES.some((prefix) => hasPathPrefix(pathname, prefix))) {
    return true
  }

  return /^\/[^/]+\/mj(?:\/|$)/.test(pathname)
}

export default {
  async fetch(request, env) {
    const requestUrl = new URL(request.url)
    if (!shouldProxyToBackend(requestUrl.pathname)) {
      const response = await env.ASSETS.fetch(request)
      if (!response.headers.get('Content-Type')?.includes('text/html')) {
        return response
      }

      const headers = new Headers(response.headers)
      headers.set('Cache-Control', 'no-cache')
      return new Response(response.body, {
        status: response.status,
        statusText: response.statusText,
        headers,
      })
    }

    let backendUrl
    try {
      backendUrl = new URL(env.BACKEND_ORIGIN)
    } catch {
      return new Response('Backend origin is not configured', { status: 503 })
    }

    if (
      backendUrl.protocol !== 'https:' ||
      backendUrl.hostname === requestUrl.hostname
    ) {
      return new Response('Backend origin is invalid', { status: 503 })
    }

    backendUrl.pathname = requestUrl.pathname
    backendUrl.search = requestUrl.search

    return fetch(new Request(backendUrl, request))
  },
}
