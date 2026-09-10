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
import { useQuery } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { z } from 'zod'

import { Button } from '@/components/ui/button'
import { api } from '@/lib/api'
import { useAuthStore } from '@/stores/auth-store'

const messageSchema = z.object({ role: z.string(), text: z.string() })
const messagesSchema = z
  .array(messageSchema)
  .transform((messages) =>
    messages.map((message, position) => ({ ...message, id: String(position) }))
  )
const contentSchema = z.object({
  input: z
    .string()
    .transform((value) => messagesSchema.parse(JSON.parse(value))),
  output: z
    .string()
    .transform((value) => messagesSchema.parse(JSON.parse(value))),
  truncated: z.boolean(),
  complete: z.boolean(),
})

type RequestContentProps = { requestId: string; isAdmin: boolean }

export function RequestContent(props: RequestContentProps) {
  const { t } = useTranslation()
  const userId = useAuthStore((state) => state.auth.user?.id)
  const query = useQuery({
    queryKey: ['request-content', userId, props.isAdmin, props.requestId],
    queryFn: async () => {
      const scope = props.isAdmin ? '' : '/self'
      const response = await api.get(
        `/api/log${scope}/content/${encodeURIComponent(props.requestId)}`
      )
      if (!response.data.success) {
        throw new Error('Unable to load request content')
      }
      return response.data.data ? contentSchema.parse(response.data.data) : null
    },
    gcTime: 0,
    retry: false,
    refetchOnWindowFocus: false,
  })

  if (query.isPending) return <p role='status'>{t('Loading...')}</p>
  if (query.isError) {
    return (
      <div role='alert'>
        {t('Unable to load request content')}{' '}
        <Button
          variant='outline'
          size='sm'
          onClick={() => void query.refetch()}
        >
          {t('Retry')}
        </Button>
      </div>
    )
  }
  if (!query.data) {
    return (
      <p className='text-muted-foreground text-xs'>
        {t('Request content was not recorded or has expired.')}
      </p>
    )
  }

  const input = query.data.input
  const lastUser = input.length - 1
  return (
    <section
      aria-label={t('Request content')}
      className='space-y-3 rounded-md border p-3 text-xs'
    >
      <h3 className='font-semibold'>{t('Request content')}</h3>
      <p className='text-muted-foreground'>
        {t(
          'Messages from this request only. Content is retained for 30 days; attachments are omitted.'
        )}
      </p>
      {query.data.truncated && (
        <p role='status'>{t('Recorded content was truncated.')}</p>
      )}
      {!query.data.complete && (
        <p role='status'>{t('A complete response was not observed.')}</p>
      )}
      {input.map((message, index) => {
        const text = (
          <p className='max-h-80 overflow-auto break-words whitespace-pre-wrap'>
            {message.text}
          </p>
        )
        if (index === lastUser && message.role === 'user') {
          return (
            <div key={message.id}>
              <h4 className='mb-1 font-medium'>{t('User question')}</h4>
              {text}
            </div>
          )
        }
        return (
          <details key={message.id} className='rounded border p-2'>
            <summary className='cursor-pointer'>
              {t('Context / tool message')} · {message.role}
            </summary>
            {text}
          </details>
        )
      })}
      {(input.length === 0 || input[lastUser].role !== 'user') && (
        <p>{t('Tool / follow-up request')}</p>
      )}
      <h4 className='font-medium'>{t('AI response')}</h4>
      {query.data.output.length === 0 ? (
        <p>{t('No response text recorded.')}</p>
      ) : (
        query.data.output.map((message) => (
          <p
            key={message.id}
            className='bg-muted/40 max-h-80 overflow-auto rounded p-2 break-words whitespace-pre-wrap'
          >
            {message.text}
          </p>
        ))
      )}
    </section>
  )
}
