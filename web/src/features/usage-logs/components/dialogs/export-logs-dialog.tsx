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
import { Csv02Icon, InformationCircleIcon } from '@hugeicons/core-free-icons'
import { HugeiconsIcon } from '@hugeicons/react'
import { useId, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { DateTimePicker } from '@/components/datetime-picker'
import { Dialog } from '@/components/dialog'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import {
  Field,
  FieldError,
  FieldGroup,
  FieldLabel,
  FieldTitle,
} from '@/components/ui/field'
import {
  Progress,
  ProgressLabel,
  ProgressValue,
} from '@/components/ui/progress'
import { Spinner } from '@/components/ui/spinner'
import { ToggleGroup, ToggleGroupItem } from '@/components/ui/toggle-group'
import dayjs from '@/lib/dayjs'

import { getAllLogs, getUserLogs } from '../../api'
import {
  buildUsageLogsCsv,
  fetchAllUsageLogPages,
  getExportPresetRange,
  validateExportRange,
  type CommonLogPageFetcher,
  type ExportProgress,
  type ExportRangePreset,
} from '../../lib/export'
import {
  buildApiParams,
  getDefaultTimeRange,
  getLogTypeConfig,
} from '../../lib/utils'

interface ExportLogsDialogProps {
  isAdmin: boolean
  searchParams: Record<string, unknown>
}

const EXPORT_PRESETS: Array<{
  value: ExportRangePreset
  label: string
}> = [
  { value: 'day', label: 'Day' },
  { value: 'week', label: 'Week' },
  { value: 'month', label: 'Month' },
  { value: 'custom', label: 'Custom' },
]

export function ExportLogsDialog(props: ExportLogsDialogProps) {
  const { t } = useTranslation()
  const rangeLabelId = useId()
  const startTimeId = useId()
  const endTimeId = useId()
  const initialRange = getDefaultTimeRange()
  const [open, setOpen] = useState(false)
  const [preset, setPreset] = useState<ExportRangePreset>('custom')
  const [startTime, setStartTime] = useState<Date | undefined>(
    initialRange.start
  )
  const [endTime, setEndTime] = useState<Date | undefined>(initialRange.end)
  const [rangeError, setRangeError] = useState<string | null>(null)
  const [exporting, setExporting] = useState(false)
  const [progress, setProgress] = useState<ExportProgress>({
    completed: 0,
    total: 0,
  })

  const handleOpenChange = (nextOpen: boolean) => {
    if (exporting && !nextOpen) return
    if (nextOpen) {
      const fallbackRange = getDefaultTimeRange()
      setStartTime(
        typeof props.searchParams.startTime === 'number'
          ? new Date(props.searchParams.startTime)
          : fallbackRange.start
      )
      setEndTime(
        typeof props.searchParams.endTime === 'number'
          ? new Date(props.searchParams.endTime)
          : fallbackRange.end
      )
      setPreset('custom')
      setRangeError(null)
      setProgress({ completed: 0, total: 0 })
    }
    setOpen(nextOpen)
  }

  const handlePresetChange = (values: unknown[]) => {
    const nextPreset = values[0]
    if (
      nextPreset !== 'day' &&
      nextPreset !== 'week' &&
      nextPreset !== 'month' &&
      nextPreset !== 'custom'
    ) {
      return
    }

    setPreset(nextPreset)
    setRangeError(null)
    if (nextPreset !== 'custom') {
      const range = getExportPresetRange(nextPreset)
      setStartTime(range.start)
      setEndTime(range.end)
    }
  }

  const handleExport = async () => {
    if (!startTime || !endTime) {
      setRangeError(t('Select both start and end times.'))
      return
    }
    const validation = validateExportRange(startTime, endTime)
    if (validation === 'invalid-order') {
      setRangeError(t('Start time must be before end time.'))
      return
    }

    setRangeError(null)
    setExporting(true)
    setProgress({ completed: 0, total: 0 })

    try {
      const exportSearchParams = {
        ...props.searchParams,
        startTime: startTime.getTime(),
        endTime: endTime.getTime(),
      }
      const params = buildApiParams({
        page: 1,
        pageSize: 100,
        searchParams: exportSearchParams,
        isAdmin: props.isAdmin,
      })
      const fetchPage: CommonLogPageFetcher = props.isAdmin
        ? (pageParams) => getAllLogs(pageParams)
        : (pageParams) => getUserLogs(pageParams)
      const logs = await fetchAllUsageLogPages({
        params,
        fetchPage,
        onProgress: setProgress,
      })

      if (logs.length === 0) {
        toast.error(t('No logs found for the selected range.'))
        return
      }

      const csv = buildUsageLogsCsv(logs, (type) =>
        t(getLogTypeConfig(type).label)
      )
      const blob = new Blob([csv], { type: 'text/csv;charset=utf-8' })
      const url = URL.createObjectURL(blob)
      const anchor = document.createElement('a')
      anchor.href = url
      anchor.download = `usage-logs-${dayjs(startTime).format('YYYYMMDD-HHmm')}-${dayjs(endTime).format('YYYYMMDD-HHmm')}.csv`
      document.body.appendChild(anchor)
      anchor.click()
      anchor.remove()
      URL.revokeObjectURL(url)

      toast.success(t('Exported {{count}} logs.', { count: logs.length }))
      setOpen(false)
    } catch (error) {
      toast.error(
        error instanceof Error && error.message
          ? error.message
          : t('Failed to export logs.')
      )
    } finally {
      setExporting(false)
    }
  }

  const progressPercent =
    progress.total > 0
      ? Math.round((progress.completed / progress.total) * 100)
      : 0

  return (
    <Dialog
      open={open}
      onOpenChange={handleOpenChange}
      trigger={
        <Button variant='outline' size='sm' aria-label={t('Export CSV')}>
          <HugeiconsIcon
            icon={Csv02Icon}
            strokeWidth={2}
            data-icon='inline-start'
          />
          <span className='hidden sm:inline'>{t('Export CSV')}</span>
        </Button>
      }
      title={t('Export usage logs')}
      description={t(
        'Choose a preset or custom date range. Current log filters are also applied.'
      )}
      contentClassName='sm:max-w-lg'
      showCloseButton={!exporting}
      footer={
        <>
          <Button
            type='button'
            variant='outline'
            onClick={() => setOpen(false)}
            disabled={exporting}
          >
            {t('Cancel')}
          </Button>
          <Button
            type='button'
            onClick={() => void handleExport()}
            disabled={exporting}
          >
            {exporting ? (
              <Spinner data-icon='inline-start' />
            ) : (
              <HugeiconsIcon
                icon={Csv02Icon}
                strokeWidth={2}
                data-icon='inline-start'
              />
            )}
            {exporting ? t('Exporting...') : t('Export CSV')}
          </Button>
        </>
      }
    >
      <FieldGroup>
        <Field>
          <FieldTitle id={rangeLabelId}>{t('Quick Range')}</FieldTitle>
          <ToggleGroup
            value={[preset]}
            onValueChange={handlePresetChange}
            variant='outline'
            spacing={2}
            aria-labelledby={rangeLabelId}
            className='grid w-full grid-cols-2 sm:grid-cols-4'
          >
            {EXPORT_PRESETS.map((option) => (
              <ToggleGroupItem key={option.value} value={option.value}>
                {t(option.label)}
              </ToggleGroupItem>
            ))}
          </ToggleGroup>
        </Field>

        <Field data-invalid={rangeError != null}>
          <FieldLabel htmlFor={startTimeId}>{t('Start Time')}</FieldLabel>
          <DateTimePicker
            id={startTimeId}
            value={startTime}
            ariaInvalid={rangeError != null}
            onChange={(date) => {
              setStartTime(date)
              setPreset('custom')
              setRangeError(null)
            }}
            placeholder={t('Select start time')}
          />
        </Field>

        <Field data-invalid={rangeError != null}>
          <FieldLabel htmlFor={endTimeId}>{t('End Time')}</FieldLabel>
          <DateTimePicker
            id={endTimeId}
            value={endTime}
            ariaInvalid={rangeError != null}
            onChange={(date) => {
              setEndTime(date)
              setPreset('custom')
              setRangeError(null)
            }}
            placeholder={t('Select end time')}
          />
          <FieldError>{rangeError}</FieldError>
        </Field>

        <Alert>
          <HugeiconsIcon icon={InformationCircleIcon} strokeWidth={2} />
          <AlertDescription>
            {t('The CSV may contain sensitive log data. Store it securely.')}
          </AlertDescription>
        </Alert>

        {exporting && (
          <Progress value={progressPercent}>
            <ProgressLabel>
              {progress.total > 0
                ? t('Exporting {{completed}} of {{total}} logs...', {
                    completed: progress.completed,
                    total: progress.total,
                  })
                : t('Preparing export...')}
            </ProgressLabel>
            <ProgressValue>{() => `${progressPercent}%`}</ProgressValue>
          </Progress>
        )}
      </FieldGroup>
    </Dialog>
  )
}
