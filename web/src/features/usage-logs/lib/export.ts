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
import dayjs from '@/lib/dayjs'

export type ExportRangePreset = 'day' | 'week' | 'month' | 'custom'

export function getExportPresetRange(
  preset: Exclude<ExportRangePreset, 'custom'>,
  now: Date = new Date()
): { start: Date; end: Date } {
  const end = new Date(now)
  const start = new Date(now)

  if (preset === 'day') {
    start.setDate(start.getDate() - 1)
  } else if (preset === 'week') {
    start.setDate(start.getDate() - 7)
  } else {
    return { start: dayjs(now).subtract(1, 'month').toDate(), end }
  }

  return { start, end }
}

export function validateExportRange(
  start: Date | undefined,
  end: Date | undefined
): 'missing' | 'invalid-order' | null {
  if (!start || !end) return 'missing'
  if (start.getTime() > end.getTime()) return 'invalid-order'
  return null
}
