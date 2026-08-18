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
  AnalyticsUpIcon,
  Key01Icon,
  Settings02Icon,
} from '@hugeicons/core-free-icons'
import { HugeiconsIcon } from '@hugeicons/react'
import { useTranslation } from 'react-i18next'

import { HeroTerminalDemo } from '../hero-terminal-demo'

const STEPS = [
  {
    number: '01',
    title: 'Configure',
    description:
      'Add your API keys, set up channels and configure access permissions',
    icon: Settings02Icon,
  },
  {
    number: '02',
    title: 'Connect',
    description:
      'Connect through OpenAI, Claude, Gemini, and other compatible API routes',
    icon: Key01Icon,
  },
  {
    number: '03',
    title: 'Monitor',
    description: 'Track usage, costs and performance with real-time analytics',
    icon: AnalyticsUpIcon,
  },
] as const

export function HowItWorks() {
  const { t } = useTranslation()

  return (
    <section className='border-b px-5 py-16 sm:px-6 md:py-20'>
      <div className='mx-auto grid max-w-6xl gap-12 lg:grid-cols-[0.82fr_1.18fr] lg:items-center'>
        <div>
          <p className='text-muted-foreground mb-2 text-xs font-medium tracking-[0.14em] uppercase'>
            {t('How It Works')}
          </p>
          <h2 className='text-2xl font-semibold tracking-tight md:text-3xl'>
            {t('Start with one API key')}
          </h2>
          <p className='text-muted-foreground mt-3 max-w-lg text-sm leading-6'>
            {t(
              'From first request to production traffic, keep every step in one gateway.'
            )}
          </p>

          <ol className='mt-8 divide-y border-y'>
            {STEPS.map((step) => (
              <li key={step.number} className='flex gap-4 py-5'>
                <div className='bg-muted flex size-10 shrink-0 items-center justify-center rounded-lg border'>
                  <HugeiconsIcon icon={step.icon} size={19} strokeWidth={1.7} />
                </div>
                <div className='min-w-0'>
                  <div className='flex items-center gap-2'>
                    <span className='text-muted-foreground font-mono text-[11px]'>
                      {step.number}
                    </span>
                    <h3 className='text-sm font-semibold'>{t(step.title)}</h3>
                  </div>
                  <p className='text-muted-foreground mt-1 text-sm leading-6'>
                    {t(step.description)}
                  </p>
                </div>
              </li>
            ))}
          </ol>
        </div>

        <HeroTerminalDemo />
      </div>
    </section>
  )
}
