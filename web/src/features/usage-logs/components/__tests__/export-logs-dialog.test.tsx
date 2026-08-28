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

import { exportUsageLogs } from '../../api'
import { ExportLogsDialog } from '../dialogs/export-logs-dialog'

vi.mock('../../api', () => ({
  exportUsageLogs: vi.fn(),
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
    Object.defineProperty(URL, 'createObjectURL', {
      configurable: true,
      value: vi.fn(() => 'blob:usage-log-export'),
    })
    Object.defineProperty(URL, 'revokeObjectURL', {
      configurable: true,
      value: vi.fn(),
    })
  })

  beforeEach(() => {
    vi.mocked(exportUsageLogs).mockReset()
    vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(() => {})
  })

  test('uses one download request and keeps export disabled while it is loading', async () => {
    let resolveRequest: ((value: Blob) => void) | undefined
    vi.mocked(exportUsageLogs).mockReturnValue(
      new Promise((resolve) => {
        resolveRequest = resolve
      })
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
    expect(exportUsageLogs).toHaveBeenCalledTimes(1)
    expect(exportUsageLogs).toHaveBeenCalledWith(
      expect.objectContaining({
        start_timestamp: new Date('2026-08-27T00:00:00Z').getTime() / 1000,
        end_timestamp: new Date('2026-08-28T00:00:00Z').getTime() / 1000,
      }),
      true
    )

    resolveRequest?.(new Blob(['csv']))
    await waitFor(() => expect(screen.queryByRole('dialog')).toBeNull())
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
    expect(exportUsageLogs).not.toHaveBeenCalled()
  })
})
