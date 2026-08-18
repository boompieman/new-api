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
import {
  ArrowRight01Icon,
  BookOpen01Icon,
  CheckmarkCircle02Icon,
} from '@hugeicons/core-free-icons'
import { HugeiconsIcon } from '@hugeicons/react'
import { Link } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { useStatus } from '@/hooks/use-status'

interface HeroProps {
  className?: string
  isAuthenticated?: boolean
}

const HERO_PROOFS = [
  'Multi-protocol Compatible',
  'Load Balancing',
  'Cost Tracking',
] as const

export function Hero(props: HeroProps) {
  const { t } = useTranslation()
  const { status } = useStatus()
  const docsUrl =
    (status?.docs_link as string | undefined) || 'https://docs.newapi.pro'
  const isExternalDocs = docsUrl.startsWith('http')

  return (
    <section className='relative border-b px-5 pt-28 pb-20 sm:px-6 md:pt-36 md:pb-24'>
      <div className='mx-auto flex max-w-4xl flex-col items-center text-center'>
        <div className='text-muted-foreground mb-5 inline-flex items-center gap-2 text-xs font-medium tracking-[0.16em] uppercase'>
          <span className='bg-foreground size-1.5 rounded-full' />
          {t('Open-source AI API gateway')}
        </div>

        <h1 className='max-w-5xl text-[clamp(2.3rem,4.5vw,3.25rem)] leading-[1.08] font-semibold tracking-[-0.04em]'>
          {t('OmniAI, connect every AI model')}
        </h1>
        <p className='text-muted-foreground mt-6 max-w-2xl text-base leading-7 md:text-lg md:leading-8'>
          {t(
            'Connect OpenAI, Claude, Gemini, and more through one API. Route traffic, manage costs, and monitor usage from a single place.'
          )}
        </p>

        <div className='mt-8 flex flex-wrap items-center justify-center gap-2.5'>
          <Button
            className='bg-foreground text-background hover:bg-foreground/85 h-10 rounded-lg px-4'
            render={
              <Link to={props.isAuthenticated ? '/wallet' : '/sign-up'} />
            }
          >
            {t(
              props.isAuthenticated
                ? 'Top up API balance'
                : 'Create free account'
            )}
            <HugeiconsIcon icon={ArrowRight01Icon} size={16} strokeWidth={2} />
          </Button>
          <Button
            variant='outline'
            className='h-10 rounded-lg px-4'
            render={<Link to={props.isAuthenticated ? '/keys' : '/pricing'} />}
          >
            {t(
              props.isAuthenticated
                ? 'Manage API keys'
                : 'View models & pricing'
            )}
          </Button>
          <Button
            variant='ghost'
            className='text-muted-foreground h-10 rounded-lg px-4'
            render={
              isExternalDocs ? (
                <a href={docsUrl} target='_blank' rel='noopener noreferrer' />
              ) : (
                <Link to={docsUrl} />
              )
            }
          >
            <HugeiconsIcon icon={BookOpen01Icon} size={16} strokeWidth={2} />
            {t('Docs')}
          </Button>
        </div>

        <div className='text-muted-foreground mt-10 flex flex-wrap items-center justify-center gap-x-5 gap-y-2 text-xs sm:text-sm'>
          {HERO_PROOFS.map((proof) => (
            <span key={proof} className='inline-flex items-center gap-1.5'>
              <HugeiconsIcon
                icon={CheckmarkCircle02Icon}
                size={15}
                strokeWidth={1.8}
                className='text-foreground/70'
              />
              {t(proof)}
            </span>
          ))}
        </div>
      </div>
    </section>
  )
}
