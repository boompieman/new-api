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
  Analytics01Icon,
  GitPullRequestIcon,
  LicenseIcon,
  SlidersHorizontalIcon,
} from '@hugeicons/core-free-icons'
import { HugeiconsIcon } from '@hugeicons/react'
import { useTranslation } from 'react-i18next'

const FEATURES = [
  {
    title: 'Developer Friendly',
    description: 'Compatible API routes for common AI application workflows',
    icon: GitPullRequestIcon,
  },
  {
    title: 'Secure & Reliable',
    description:
      'Enterprise-grade security with comprehensive permission management',
    icon: SlidersHorizontalIcon,
  },
  {
    title: 'Transparent Billing',
    description: 'Pay-as-you-go with real-time usage monitoring',
    icon: Analytics01Icon,
  },
  {
    title: 'Open Source',
    description: 'Community driven, self-hosted, and extensible',
    icon: LicenseIcon,
  },
] as const

interface FeaturesProps {
  className?: string
}

export function Features(_props: FeaturesProps) {
  const { t } = useTranslation()

  return (
    <section className='border-b px-5 py-16 sm:px-6 md:py-20'>
      <div className='mx-auto max-w-6xl'>
        <div className='mb-9 max-w-2xl'>
          <p className='text-muted-foreground mb-2 text-xs font-medium tracking-[0.14em] uppercase'>
            {t('Core Features')}
          </p>
          <h2 className='text-2xl font-semibold tracking-tight md:text-3xl'>
            {t('Built for production AI traffic')}
          </h2>
          <p className='text-muted-foreground mt-3 text-sm leading-6'>
            {t(
              'Everything you need to route, control, and understand model usage.'
            )}
          </p>
        </div>

        <div className='grid overflow-hidden rounded-lg border sm:grid-cols-2 lg:grid-cols-4'>
          {FEATURES.map((feature, index) => (
            <article
              key={feature.title}
              className='border-b p-6 last:border-b-0 sm:border-r lg:border-b-0 sm:[&:nth-child(2)]:border-r-0 lg:[&:nth-child(2)]:border-r'
            >
              <div className='bg-muted mb-5 flex size-10 items-center justify-center rounded-lg border'>
                <HugeiconsIcon
                  icon={feature.icon}
                  size={19}
                  strokeWidth={1.7}
                />
              </div>
              <span className='text-muted-foreground font-mono text-[11px]'>
                0{index + 1}
              </span>
              <h3 className='mt-2 text-sm font-semibold'>{t(feature.title)}</h3>
              <p className='text-muted-foreground mt-2 text-sm leading-6'>
                {t(feature.description)}
              </p>
            </article>
          ))}
        </div>
      </div>
    </section>
  )
}
