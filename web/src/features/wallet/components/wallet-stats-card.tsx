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
import { Link } from '@tanstack/react-router'
import { Activity, Receipt } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { formatQuota } from '@/lib/format'

import type { UserWalletData } from '../types'

interface WalletStatsCardProps {
  user: UserWalletData | null
  loading?: boolean
  onOpenBilling: () => void
}

export function WalletStatsCard(props: WalletStatsCardProps) {
  const { t } = useTranslation()

  return (
    <Card data-card-hover='false' className='gap-0 py-0'>
      <CardContent className='flex flex-col gap-5 p-5 sm:p-6 md:flex-row md:items-end md:justify-between'>
        <div className='min-w-0'>
          <div className='text-muted-foreground text-sm font-medium'>
            {t('Current Balance')}
          </div>
          {props.loading ? (
            <Skeleton className='mt-3 h-11 w-48' />
          ) : (
            <div className='mt-2 font-mono text-4xl font-bold tracking-tight break-all tabular-nums sm:text-5xl'>
              {formatQuota(props.user?.quota ?? 0)}
            </div>
          )}
          <div className='text-muted-foreground mt-2 text-sm'>
            {t('Remaining quota')}
          </div>
        </div>

        <div className='flex flex-wrap items-center gap-2'>
          <Button
            variant='ghost'
            render={
              <Link to='/dashboard/$section' params={{ section: 'overview' }} />
            }
          >
            <Activity data-icon='inline-start' />
            {t('Usage')}
          </Button>
          <Button variant='outline' onClick={props.onOpenBilling}>
            <Receipt data-icon='inline-start' />
            {t('Order History')}
          </Button>
        </div>
      </CardContent>
    </Card>
  )
}
