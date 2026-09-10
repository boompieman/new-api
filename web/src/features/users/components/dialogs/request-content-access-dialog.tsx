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
import { useTranslation } from 'react-i18next'

import { Dialog } from '@/components/dialog'
import { Switch } from '@/components/ui/switch'
import { api } from '@/lib/api'
import { handleServerError } from '@/lib/handle-server-error'

interface RequestContentAccessDialogProps {
  userId: number
  username: string
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function RequestContentAccessDialog(
  props: RequestContentAccessDialogProps
) {
  const { t } = useTranslation()
  const client = useQueryClient()
  const key = ['request-content-access', props.userId]
  const path = `/api/user/${props.userId}/request-content-access`
  const query = useQuery({
    queryKey: key,
    queryFn: async () => {
      const response = await api.get(path)
      if (!response.data.success) {
        throw new Error(response.data.message)
      }
      return response.data.data.enabled as boolean
    },
    enabled: props.open,
    gcTime: 0,
  })
  const mutation = useMutation({
    mutationFn: async (enabled: boolean) => {
      const response = await api.put(path, { enabled })
      if (!response.data.success) {
        throw new Error(response.data.message)
      }
    },
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: key })
    },
    onError: (error) => handleServerError(error),
  })
  return (
    <Dialog
      open={props.open}
      onOpenChange={props.onOpenChange}
      title={t('Request content access')}
      description={props.username}
      contentHeight='auto'
    >
      <div className='space-y-4 py-2'>
        <p className='text-muted-foreground text-sm'>
          {t(
            'Recording continues for all accounts. This switch only allows this user to view their own request content.'
          )}
        </p>
        {query.isError ? (
          <p role='alert'>{t('Unable to load request content')}</p>
        ) : (
          <label className='flex items-center justify-between gap-3 text-sm'>
            {t('Allow this user to view request content')}
            <Switch
              checked={query.data ?? false}
              disabled={query.isPending || mutation.isPending}
              onCheckedChange={(enabled) => mutation.mutate(enabled)}
            />
          </label>
        )}
      </div>
    </Dialog>
  )
}
