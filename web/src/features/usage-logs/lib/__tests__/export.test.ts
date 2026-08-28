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
import { describe, expect, test, vi } from 'vitest'

import type { UsageLog } from '../../data/schema'
import type { GetLogsResponse } from '../../types'
import {
  buildUsageLogsCsv,
  fetchAllUsageLogPages,
  getExportPresetRange,
  validateExportRange,
} from '../export'

function createLog(id: number, content = ''): UsageLog {
  return {
    id,
    user_id: 3,
    created_at: 1_700_000_000 + id,
    type: 2,
    content,
    username: 'tester',
    token_name: 'default',
    model_name: 'gpt-test',
    quota: 100,
    prompt_tokens: 10,
    completion_tokens: 5,
    use_time: 2,
    is_stream: true,
    channel: 1,
    channel_name: 'primary',
    token_id: 4,
    group: 'default',
    ip: '127.0.0.1',
    other: '{}',
    request_id: `request-${id}`,
    upstream_request_id: '',
  }
}

function createPage(items: UsageLog[], total: number): GetLogsResponse {
  return {
    success: true,
    data: {
      items,
      total,
      page: 1,
      page_size: 100,
    },
  }
}

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

describe('usage log CSV generation', () => {
  test('writes UTF-8 CSV with stable fields and escaped multiline values', () => {
    const csv = buildUsageLogsCsv(
      [createLog(1, 'quoted "value"\nsecond line')],
      () => 'Consume'
    )

    expect(csv.startsWith('\uFEFF')).toBe(true)
    expect(csv).toContain('"created_time"')
    expect(csv).toContain('"quoted ""value""\nsecond line"')
    expect(csv).toContain(',15,100,2,true,')
  })

  test('neutralizes spreadsheet formulas in text fields', () => {
    const csv = buildUsageLogsCsv(
      [createLog(1, '=HYPERLINK("bad")')],
      () => '+SUM(A1:A2)'
    )

    expect(csv).toContain('"\'+SUM(A1:A2)"')
    expect(csv).toContain('"\'=HYPERLINK(""bad"")"')
  })
})

describe('usage log page collection', () => {
  test('fetches every page in bounded batches and reports progress', async () => {
    const pages = new Map([
      [
        1,
        createPage(
          Array.from({ length: 100 }, (_, index) => createLog(index + 1)),
          205
        ),
      ],
      [
        2,
        createPage(
          Array.from({ length: 100 }, (_, index) => createLog(index + 101)),
          205
        ),
      ],
      [
        3,
        createPage(
          Array.from({ length: 5 }, (_, index) => createLog(index + 201)),
          205
        ),
      ],
    ])
    const fetchPage = vi.fn(async (params) => {
      const response = pages.get(params.p ?? 1)
      if (!response) throw new Error('missing page fixture')
      return response
    })
    const onProgress = vi.fn()

    const logs = await fetchAllUsageLogPages({
      params: { type: 2 },
      fetchPage,
      onProgress,
    })

    expect(logs).toHaveLength(205)
    expect(fetchPage).toHaveBeenCalledTimes(3)
    expect(fetchPage).toHaveBeenNthCalledWith(
      1,
      expect.objectContaining({ p: 1, page_size: 100, type: 2 })
    )
    expect(onProgress).toHaveBeenLastCalledWith({ completed: 205, total: 205 })
  })

  test('stops the export when any page reports an API error', async () => {
    const fetchPage = vi
      .fn()
      .mockResolvedValueOnce(createPage([createLog(1)], 101))
      .mockResolvedValueOnce({ success: false, message: 'permission denied' })

    await expect(
      fetchAllUsageLogPages({ params: {}, fetchPage })
    ).rejects.toThrow('permission denied')
  })
})
