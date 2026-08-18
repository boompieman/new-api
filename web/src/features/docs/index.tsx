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
import { ArrowRight01Icon } from '@hugeicons/core-free-icons'
import { HugeiconsIcon } from '@hugeicons/react'
import { Link } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'

import { PublicLayout } from '@/components/layout/components/public-layout'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { useStatus } from '@/hooks/use-status'
import { useSystemConfig } from '@/hooks/use-system-config'

import { ApiReferenceSection } from './components/api-reference-section'
import { CodeExample } from './components/code-example'
import { DocsNavigation, DocsTopicBar } from './components/docs-navigation'
import { QuickStartSection } from './components/quick-start-section'

function getApiOrigin(serverAddress: unknown): string {
  const currentOrigin = window.location.origin.replace(/\/$/, '')
  if (typeof serverAddress !== 'string' || serverAddress.trim() === '') {
    return currentOrigin
  }

  try {
    const configuredUrl = new URL(serverAddress.trim())
    if (
      configuredUrl.hostname === 'localhost' ||
      configuredUrl.hostname === '127.0.0.1' ||
      configuredUrl.hostname === '[::1]'
    ) {
      return currentOrigin
    }
    return configuredUrl.origin
  } catch {
    return currentOrigin
  }
}

export function Docs() {
  const { t } = useTranslation()
  const { status } = useStatus()
  const { systemName } = useSystemConfig()
  const displayName = systemName || 'omniAI'
  const serverAddress = status?.server_address ?? status?.data?.server_address
  const apiOrigin = getApiOrigin(serverAddress)
  const baseUrl = `${apiOrigin}/v1`

  return (
    <PublicLayout showMainContainer={false}>
      <main className='pt-16'>
        <DocsTopicBar />
        <div className='mx-auto grid max-w-7xl grid-cols-1 gap-10 px-4 lg:grid-cols-[13rem_minmax(0,1fr)] lg:px-6 xl:grid-cols-[13rem_minmax(0,1fr)_12rem]'>
          <aside className='sticky top-28 hidden h-[calc(100svh-8rem)] overflow-y-auto py-10 lg:block'>
            <DocsNavigation />
          </aside>

          <article className='max-w-3xl min-w-0 py-10 lg:py-14'>
            <section id='overview' className='scroll-mt-32'>
              <p className='text-primary text-sm font-semibold'>
                {t('Developer documentation')}
              </p>
              <h1 className='mt-3 text-4xl font-bold tracking-tight text-balance sm:text-5xl'>
                {t('Build with {{name}}', { name: displayName })}
              </h1>
              <p className='text-muted-foreground mt-5 max-w-2xl text-lg leading-8'>
                {t(
                  'One OpenAI-compatible endpoint for chat, responses, embeddings, images, audio, and more.'
                )}
              </p>

              <div className='mt-8 grid gap-3 sm:grid-cols-2'>
                <Card size='sm'>
                  <CardHeader>
                    <CardTitle>{t('Quick start')}</CardTitle>
                    <CardDescription>
                      {t('Make your first API request in about five minutes.')}
                    </CardDescription>
                  </CardHeader>
                  <CardContent>
                    <a
                      href='#quick-start'
                      className='text-primary inline-flex items-center gap-1 text-sm font-medium hover:underline'
                    >
                      {t('Start building')}
                      <HugeiconsIcon
                        icon={ArrowRight01Icon}
                        className='size-4'
                      />
                    </a>
                  </CardContent>
                </Card>
                <Card size='sm'>
                  <CardHeader>
                    <CardTitle>{t('API endpoints')}</CardTitle>
                    <CardDescription>
                      {t('Explore compatible routes for common AI workflows.')}
                    </CardDescription>
                  </CardHeader>
                  <CardContent>
                    <a
                      href='#endpoints'
                      className='text-primary inline-flex items-center gap-1 text-sm font-medium hover:underline'
                    >
                      {t('View API reference')}
                      <HugeiconsIcon
                        icon={ArrowRight01Icon}
                        className='size-4'
                      />
                    </a>
                  </CardContent>
                </Card>
              </div>

              <div className='mt-8'>
                <p className='text-muted-foreground mb-2 text-sm font-medium'>
                  {t('Your API base URL')}
                </p>
                <CodeExample code={baseUrl} label='Base URL' />
              </div>

              <div className='mt-8 flex flex-wrap gap-3'>
                <Button nativeButton={false} render={<Link to='/keys' />}>
                  {t('Create API key')}
                </Button>
                <Button
                  nativeButton={false}
                  variant='outline'
                  render={<Link to='/pricing' />}
                >
                  {t('Browse models and pricing')}
                </Button>
              </div>
            </section>

            <QuickStartSection apiOrigin={apiOrigin} />
            <ApiReferenceSection apiOrigin={apiOrigin} />
          </article>

          <aside className='sticky top-28 hidden h-fit py-14 xl:block'>
            <p className='text-sm font-semibold'>{t('On this page')}</p>
            <div className='text-muted-foreground mt-3 flex flex-col gap-2 text-sm'>
              <a href='#quick-start' className='hover:text-foreground'>
                {t('Quick start')}
              </a>
              <a href='#authentication' className='hover:text-foreground'>
                {t('Authentication')}
              </a>
              <a href='#protocols' className='hover:text-foreground'>
                {t('API protocols')}
              </a>
              <a href='#capabilities' className='hover:text-foreground'>
                {t('Core capabilities')}
              </a>
              <a href='#endpoints' className='hover:text-foreground'>
                {t('API endpoints')}
              </a>
              <a href='#errors' className='hover:text-foreground'>
                {t('Error handling')}
              </a>
              <a href='#checklist' className='hover:text-foreground'>
                {t('Integration checklist')}
              </a>
            </div>
          </aside>
        </div>
      </main>
    </PublicLayout>
  )
}
