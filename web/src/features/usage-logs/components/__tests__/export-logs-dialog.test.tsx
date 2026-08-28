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
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import i18next from 'i18next'
import { beforeAll, beforeEach, describe, expect, test, vi } from 'vitest'

import { getAllLogs } from '../../api'
import type { GetLogsResponse } from '../../types'
import { ExportLogsDialog } from '../dialogs/export-logs-dialog'

vi.mock('../../api', () => ({
  getAllLogs: vi.fn(),
  getUserLogs: vi.fn(),
}))

vi.mock('sonner', () => ({
  toast: {
    error: vi.fn(),
    success: vi.fn(),
  },
}))

describe('usage log export dialog', () => {
  beforeAll(() => {
    i18next.addResourceBundle('en', 'translation', {
      'Export CSV': 'Export CSV',
      'Export usage logs': 'Export usage logs',
      'Preparing export...': 'Preparing export...',
      'Select both start and end times.': 'Select both start and end times.',
    })
  })

  beforeEach(() => {
    vi.mocked(getAllLogs).mockReset()
  })

  test('keeps export disabled while the first page is loading', async () => {
    let resolveRequest: ((value: GetLogsResponse) => void) | undefined
    vi.mocked(getAllLogs).mockReturnValue(
      new Promise((resolve) => {
        resolveRequest = resolve
      }) as ReturnType<typeof getAllLogs>
    )
    const user = userEvent.setup()

    render(
      <ExportLogsDialog
        isAdmin
        searchParams={{
          startTime: new Date('2026-08-27T00:00:00Z').getTime(),
          endTime: new Date('2026-08-28T00:00:00Z').getTime(),
        }}
      />
    )

    await user.click(screen.getByRole('button', { name: 'Export CSV' }))
    const dialog = screen.getByRole('dialog')
    const exportButton = within(dialog).getByRole('button', {
      name: 'Export CSV',
    })
    await user.click(exportButton)

    expect(exportButton).toBeDisabled()
    expect(within(dialog).getByText('Preparing export...')).toBeVisible()

    resolveRequest?.({
      success: true,
      data: { items: [], total: 0, page: 1, page_size: 100 },
    })
    await waitFor(() => expect(exportButton).toBeEnabled())
  })

  test('shows a validation error without requesting data when a date is missing', async () => {
    const user = userEvent.setup()
    render(<ExportLogsDialog isAdmin searchParams={{}} />)

    await user.click(screen.getByRole('button', { name: 'Export CSV' }))
    const dialog = screen.getByRole('dialog')
    await user.click(
      within(dialog).getAllByRole('button', { name: 'Clear' })[0]
    )
    await user.click(within(dialog).getByRole('button', { name: 'Export CSV' }))

    expect(
      within(dialog).getByText('Select both start and end times.')
    ).toBeVisible()
    expect(getAllLogs).not.toHaveBeenCalled()
  })
})
