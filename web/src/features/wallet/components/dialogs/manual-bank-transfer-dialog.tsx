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
import { Copy01Icon, Tick02Icon } from '@hugeicons/core-free-icons'
import { HugeiconsIcon } from '@hugeicons/react'
import { useTranslation } from 'react-i18next'

import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Separator } from '@/components/ui/separator'
import { useCopyToClipboard } from '@/hooks/use-copy-to-clipboard'

import { formatCurrency } from '../../lib'
import type { ManualBankTransferDetails } from '../../types'

function TransferDetailRow({
  label,
  value,
  copyValue,
  copiedText,
  onCopy,
}: {
  label: string
  value: string
  copyValue?: string
  copiedText: string | null
  onCopy: (value: string) => void
}) {
  const { t } = useTranslation()

  return (
    <div className='flex min-w-0 items-center justify-between gap-3 py-2'>
      <span className='text-muted-foreground shrink-0 text-sm'>{label}</span>
      <div className='flex min-w-0 items-center justify-end gap-1.5'>
        <span className='text-right font-medium break-all'>{value}</span>
        {copyValue && (
          <Button
            type='button'
            variant='ghost'
            size='icon-sm'
            aria-label={t('Copy {{label}}', { label })}
            onClick={() => onCopy(copyValue)}
          >
            <HugeiconsIcon
              icon={copiedText === copyValue ? Tick02Icon : Copy01Icon}
              strokeWidth={2}
            />
          </Button>
        )}
      </div>
    </div>
  )
}

export function ManualBankTransferDialog({
  details,
  onClose,
}: {
  details: ManualBankTransferDetails | null
  onClose: () => void
}) {
  const { t } = useTranslation()
  const { copiedText, copyToClipboard } = useCopyToClipboard()

  if (!details) return null

  const bankDisplay = [details.bank_name, details.bank_code]
    .filter(Boolean)
    .join(' ')

  return (
    <Dialog open onOpenChange={(open) => !open && onClose()}>
      <DialogContent className='max-h-[calc(100dvh-2rem)] overflow-y-auto sm:max-w-lg'>
        <DialogHeader>
          <div className='flex items-center gap-2'>
            <DialogTitle>{t('Bank transfer details')}</DialogTitle>
            <Badge variant='secondary'>{t('Pending confirmation')}</Badge>
          </div>
          <DialogDescription>
            {t(
              'Transfer the exact amount below. Your balance will be added after the administrator confirms the payment.'
            )}
          </DialogDescription>
        </DialogHeader>

        <div>
          <TransferDetailRow
            label={t('Amount to transfer')}
            value={formatCurrency(details.payment_amount)}
            copyValue={String(details.payment_amount)}
            copiedText={copiedText}
            onCopy={copyToClipboard}
          />
          <Separator />
          <TransferDetailRow
            label={t('Bank')}
            value={bankDisplay}
            copiedText={copiedText}
            onCopy={copyToClipboard}
          />
          {details.branch_name && (
            <>
              <Separator />
              <TransferDetailRow
                label={t('Branch')}
                value={details.branch_name}
                copiedText={copiedText}
                onCopy={copyToClipboard}
              />
            </>
          )}
          <Separator />
          <TransferDetailRow
            label={t('Account name')}
            value={details.account_name}
            copiedText={copiedText}
            onCopy={copyToClipboard}
          />
          <Separator />
          <TransferDetailRow
            label={t('Account number')}
            value={details.account_number}
            copyValue={details.account_number}
            copiedText={copiedText}
            onCopy={copyToClipboard}
          />
          <Separator />
          <TransferDetailRow
            label={t('Order number')}
            value={details.order_id}
            copyValue={details.order_id}
            copiedText={copiedText}
            onCopy={copyToClipboard}
          />
        </div>

        <Alert>
          <AlertTitle>{t('After transferring')}</AlertTitle>
          <AlertDescription className='whitespace-pre-wrap'>
            {details.instructions ||
              t('Contact the administrator and provide your order number.')}
          </AlertDescription>
        </Alert>

        <DialogFooter>
          <Button type='button' onClick={onClose}>
            {t('Done')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
