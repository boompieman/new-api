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
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Skeleton } from '@/components/ui/skeleton'
import { Switch } from '@/components/ui/switch'

import { SettingsSection } from '../components/settings-section'
import {
  getPinnacleToolConfig,
  testPinnacleToolConnection,
  updatePinnacleToolConfig,
} from './pinnacle-tool-api'

const queryKey = ['system-settings', 'pinnacle-tool'] as const

export function PinnacleToolSettingsSection() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [enabled, setEnabled] = useState(false)
  const [apiToken, setApiToken] = useState('')

  const configQuery = useQuery({
    queryKey,
    queryFn: getPinnacleToolConfig,
  })

  useEffect(() => {
    if (configQuery.data) setEnabled(configQuery.data.enabled)
  }, [configQuery.data])

  const saveMutation = useMutation({
    mutationFn: updatePinnacleToolConfig,
    onSuccess: async () => {
      setApiToken('')
      await queryClient.invalidateQueries({ queryKey })
      toast.success(t('Pinnacle API tool configuration saved'))
    },
  })

  const testMutation = useMutation({
    mutationFn: testPinnacleToolConnection,
    onSuccess: () => toast.success(t('Pinnacle API connection verified')),
  })

  if (configQuery.isLoading) {
    return <Skeleton className='h-64 w-full' />
  }

  const config = configQuery.data

  return (
    <SettingsSection title={t('Pinnacle API Tool')}>
      <div className='border-border flex flex-col gap-6 rounded-xl border p-5'>
        <div className='flex flex-wrap items-start justify-between gap-3'>
          <div>
            <div className='flex items-center gap-2'>
              <h4 className='font-medium'>{t('Pinnacle sports data')}</h4>
              <Badge variant={config?.token_configured ? 'default' : 'outline'}>
                {config?.token_configured
                  ? t('Credential configured')
                  : t('Credential required')}
              </Badge>
            </div>
            <p className='text-muted-foreground mt-1 text-sm'>
              {t(
                'Expose Pinnacle sports, league, match, live match, and odds data through authenticated OmniAI endpoints.'
              )}
            </p>
          </div>
          <div className='text-right'>
            <p className='font-semibold'>$5 USD / 1,000</p>
            <p className='text-muted-foreground text-xs'>
              {t('successful requests')}
            </p>
          </div>
        </div>

        <div className='flex items-center justify-between gap-4'>
          <div>
            <Label htmlFor='pinnacle-enabled'>{t('Enable API tool')}</Label>
            <p className='text-muted-foreground mt-1 text-xs'>
              {t('Customers authenticate with their existing OmniAI API key.')}
            </p>
          </div>
          <Switch
            id='pinnacle-enabled'
            checked={enabled}
            onCheckedChange={setEnabled}
          />
        </div>

        <div className='space-y-2'>
          <Label htmlFor='pinnacle-token'>{t('Apify API token')}</Label>
          <Input
            id='pinnacle-token'
            type='password'
            autoComplete='new-password'
            value={apiToken}
            onChange={(event) => setApiToken(event.target.value)}
            placeholder={
              config?.token_configured
                ? t('Leave blank to keep the saved token')
                : t('Paste the Apify token')
            }
          />
          <p className='text-muted-foreground text-xs'>
            {t('The saved token is never returned to the browser.')}
          </p>
        </div>

        <div className='flex flex-wrap justify-end gap-2'>
          <Button
            variant='outline'
            disabled={!config?.token_configured || testMutation.isPending}
            onClick={() => testMutation.mutate()}
          >
            {testMutation.isPending ? t('Testing...') : t('Test connection')}
          </Button>
          <Button
            disabled={saveMutation.isPending}
            onClick={() =>
              saveMutation.mutate({
                enabled,
                apiToken: apiToken.trim() || undefined,
              })
            }
          >
            {saveMutation.isPending ? t('Saving...') : t('Save')}
          </Button>
        </div>
      </div>
    </SettingsSection>
  )
}
