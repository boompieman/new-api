/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { afterEach, expect, test, vi } from 'vitest'

import { api } from '@/lib/api'

import { RequestContent } from '../request-content'

const clients: QueryClient[] = []
function renderContent() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  clients.push(client)
  return render(
    <QueryClientProvider client={client}>
      <RequestContent requestId='req-1' isAdmin={false} />
    </QueryClientProvider>
  )
}
afterEach(() => {
  clients.forEach((client) => client.clear())
  clients.length = 0
  vi.restoreAllMocks()
})

test('shows request question and answer while keeping context collapsed and treating text as text', async () => {
  const get = vi.spyOn(api, 'get').mockResolvedValue({
    data: {
      success: true,
      data: {
        input: JSON.stringify([
          { role: 'system', text: 'private context' },
          { role: 'user', text: '<script>alert(1)</script>' },
        ]),
        output: JSON.stringify([{ role: 'assistant', text: 'answer' }]),
        truncated: true,
        complete: false,
      },
    },
  })
  renderContent()
  expect(await screen.findByText('answer')).toBeVisible()
  expect(screen.getByText('<script>alert(1)</script>')).toBeVisible()
  expect(document.querySelector('script')).toBeNull()
  expect(
    screen.getByText('private context').closest('details')
  ).not.toHaveAttribute('open')
  expect(screen.getByText('Recorded content was truncated.')).toBeVisible()
  expect(
    screen.getByText('A complete response was not observed.')
  ).toBeVisible()
  expect(get).toHaveBeenCalledWith('/api/log/self/content/req-1')
})

test('shows missing or expired state without inventing a conversation', async () => {
  vi.spyOn(api, 'get').mockResolvedValue({
    data: { success: true, data: null },
  })
  renderContent()
  expect(
    await screen.findByText('Request content was not recorded or has expired.')
  ).toBeVisible()
})

test('allows retry after request failure and labels a tool continuation', async () => {
  vi.spyOn(api, 'get')
    .mockRejectedValueOnce(new Error('network'))
    .mockResolvedValueOnce({
      data: {
        success: true,
        data: {
          input: '[{"role":"tool","text":"result"}]',
          output: '[]',
          truncated: false,
          complete: false,
        },
      },
    })
  renderContent()
  await screen.findByRole('alert')
  fireEvent.click(screen.getByRole('button', { name: 'Retry' }))
  await waitFor(() =>
    expect(screen.getByText('Tool / follow-up request')).toBeVisible()
  )
  expect(screen.queryByText('User question')).not.toBeInTheDocument()
})
