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
import { expect, test, vi } from 'vitest'

import { api } from '@/lib/api'

import { RequestContentAccessDialog } from '../request-content-access-dialog'

test('administrator can grant and revoke viewing without changing capture', async () => {
  let enabled = false
  const get = vi
    .spyOn(api, 'get')
    .mockImplementation(async () => ({
      data: { success: true, data: { enabled } },
    }))
  const put = vi.spyOn(api, 'put').mockImplementation(async (_path, body) => {
    enabled = (body as { enabled: boolean }).enabled
    return { data: { success: true } }
  })
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  const view = render(
    <QueryClientProvider client={client}>
      <RequestContentAccessDialog
        userId={7}
        username='customer'
        open
        onOpenChange={() => undefined}
      />
    </QueryClientProvider>
  )
  try {
    const toggle = await screen.findByRole('switch', {
      name: 'Allow this user to view request content',
    })
    await waitFor(() => expect(toggle).toBeEnabled())
    expect(toggle).not.toBeChecked()
    fireEvent.click(toggle)
    await waitFor(() => expect(toggle).toBeChecked())
    expect(put).toHaveBeenLastCalledWith('/api/user/7/request-content-access', {
      enabled: true,
    })
    await waitFor(() => expect(toggle).toBeEnabled())
    fireEvent.click(toggle)
    await waitFor(() => expect(toggle).not.toBeChecked())
    expect(put).toHaveBeenLastCalledWith('/api/user/7/request-content-access', {
      enabled: false,
    })
  } finally {
    view.unmount()
    client.clear()
    get.mockRestore()
    put.mockRestore()
  }
})
