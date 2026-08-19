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
import i18next from 'i18next'
import { useCallback, useState } from 'react'
import { toast } from 'sonner'

import { isApiSuccess, requestManualBankTransferPayment } from '../api'
import type { ManualBankTransferDetails } from '../types'

export function useManualBankTransfer() {
  const [processing, setProcessing] = useState(false)
  const [details, setDetails] = useState<ManualBankTransferDetails | null>(null)

  const processManualBankTransfer = useCallback(async (topupAmount: number) => {
    try {
      setProcessing(true)
      const response = await requestManualBankTransferPayment({
        amount: Math.floor(topupAmount),
      })
      if (!isApiSuccess(response) || !response.data) {
        toast.error(
          response.message || i18next.t('Failed to create bank transfer order')
        )
        return false
      }

      setDetails(response.data)
      toast.success(i18next.t('Bank transfer order created'))
      return true
    } catch {
      toast.error(i18next.t('Failed to create bank transfer order'))
      return false
    } finally {
      setProcessing(false)
    }
  }, [])

  const clearDetails = useCallback(() => setDetails(null), [])

  return {
    processing,
    details,
    processManualBankTransfer,
    clearDetails,
  }
}
