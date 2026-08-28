/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { describe, expect, test } from 'vitest'

import { getExportPresetRange, validateExportRange } from '../export'

describe('usage log export ranges', () => {
  const now = new Date('2026-08-28T12:30:00.000Z')

  test('uses rolling day and week ranges ending at the supplied time', () => {
    const day = getExportPresetRange('day', now)
    const week = getExportPresetRange('week', now)

    expect(day.end).toEqual(now)
    expect(day.start.toISOString()).toBe('2026-08-27T12:30:00.000Z')
    expect(week.start.toISOString()).toBe('2026-08-21T12:30:00.000Z')
  })

  test('uses one calendar month for the monthly preset', () => {
    const month = getExportPresetRange('month', now)
    const endOfMonth = getExportPresetRange(
      'month',
      new Date('2026-03-31T12:30:00.000Z')
    )

    expect(month.start.toISOString()).toBe('2026-07-28T12:30:00.000Z')
    expect(month.end).toEqual(now)
    expect(endOfMonth.start.toISOString()).toBe('2026-02-28T12:30:00.000Z')
  })

  test('rejects missing and reversed custom ranges', () => {
    expect(validateExportRange(undefined, now)).toBe('missing')
    expect(validateExportRange(now, new Date('2026-08-27T12:30:00.000Z'))).toBe(
      'invalid-order'
    )
    expect(validateExportRange(now, now)).toBeNull()
  })
})
