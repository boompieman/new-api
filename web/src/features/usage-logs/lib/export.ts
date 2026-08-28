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

import type { UsageLog } from '../data/schema'
import type { GetLogsParams, GetLogsResponse } from '../types'

export type ExportRangePreset = 'day' | 'week' | 'month' | 'custom'

export interface ExportProgress {
  completed: number
  total: number
}

export type CommonLogPageFetcher = (
  params: GetLogsParams
) => Promise<GetLogsResponse>

const EXPORT_PAGE_SIZE = 100
const EXPORT_CONCURRENCY = 4

const CSV_HEADERS = [
  'id',
  'created_at',
  'created_time',
  'type',
  'type_name',
  'channel',
  'channel_name',
  'user_id',
  'username',
  'token_id',
  'token_name',
  'model_name',
  'group',
  'prompt_tokens',
  'completion_tokens',
  'total_tokens',
  'quota',
  'use_time',
  'is_stream',
  'ip',
  'request_id',
  'upstream_request_id',
  'content',
  'other',
] as const

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

function getUsageLogItems(response: GetLogsResponse): UsageLog[] {
  if (!response.success) {
    throw new Error(response.message || 'Failed to export logs.')
  }

  return (response.data?.items ?? []) as UsageLog[]
}

export async function fetchAllUsageLogPages(config: {
  params: GetLogsParams
  fetchPage: CommonLogPageFetcher
  onProgress?: (progress: ExportProgress) => void
}): Promise<UsageLog[]> {
  const firstResponse = await config.fetchPage({
    ...config.params,
    p: 1,
    page_size: EXPORT_PAGE_SIZE,
  })
  const firstItems = getUsageLogItems(firstResponse)
  const responseTotal = firstResponse.data?.total ?? firstItems.length
  const total = Math.max(0, responseTotal)
  const logs = [...firstItems]
  config.onProgress?.({ completed: Math.min(logs.length, total), total })

  const pageCount = Math.ceil(total / EXPORT_PAGE_SIZE)
  for (
    let firstPage = 2;
    firstPage <= pageCount;
    firstPage += EXPORT_CONCURRENCY
  ) {
    const lastPage = Math.min(firstPage + EXPORT_CONCURRENCY - 1, pageCount)
    const pageNumbers = Array.from(
      { length: lastPage - firstPage + 1 },
      (_, index) => firstPage + index
    )
    const responses = await Promise.all(
      pageNumbers.map((page) =>
        config.fetchPage({
          ...config.params,
          p: page,
          page_size: EXPORT_PAGE_SIZE,
        })
      )
    )

    for (const response of responses) {
      logs.push(...getUsageLogItems(response))
    }
    config.onProgress?.({ completed: Math.min(logs.length, total), total })
  }

  return logs.slice(0, total)
}

function escapeCsvCell(value: boolean | number | string): string {
  if (typeof value === 'number') {
    return Number.isFinite(value) ? String(value) : ''
  }
  if (typeof value === 'boolean') return value ? 'true' : 'false'

  const formulaSafeValue = /^[=+\-@\t\r]/.test(value) ? `'${value}` : value
  return `"${formulaSafeValue.replaceAll('"', '""')}"`
}

export function buildUsageLogsCsv(
  logs: UsageLog[],
  getTypeName: (type: number) => string
): string {
  const rows = logs.map((log) => [
    log.id,
    log.created_at,
    dayjs.unix(log.created_at).format('YYYY-MM-DD HH:mm:ss'),
    log.type,
    getTypeName(log.type),
    log.channel,
    log.channel_name ?? '',
    log.user_id,
    log.username,
    log.token_id,
    log.token_name,
    log.model_name,
    log.group,
    log.prompt_tokens,
    log.completion_tokens,
    log.prompt_tokens + log.completion_tokens,
    log.quota,
    log.use_time,
    log.is_stream,
    log.ip,
    log.request_id,
    log.upstream_request_id,
    log.content,
    log.other,
  ])

  const lines = [
    CSV_HEADERS.map(escapeCsvCell).join(','),
    ...rows.map((row) => row.map(escapeCsvCell).join(',')),
  ]
  return `\uFEFF${lines.join('\r\n')}\r\n`
}
