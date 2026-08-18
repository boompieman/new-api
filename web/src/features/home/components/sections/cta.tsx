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

import { Button } from '@/components/ui/button'

interface CTAProps {
  className?: string
  isAuthenticated?: boolean
}

export function CTA(props: CTAProps) {
  const { t } = useTranslation()

  return (
    <section className='px-5 py-16 sm:px-6 md:py-20'>
      <div className='bg-foreground text-background mx-auto flex max-w-6xl flex-col justify-between gap-8 rounded-xl px-6 py-9 sm:px-9 md:flex-row md:items-center md:py-10'>
        <div>
          <h2 className='text-2xl font-semibold tracking-tight md:text-3xl'>
            {t('Ready to connect your AI stack?')}
          </h2>
          <p className='text-background/65 mt-2 text-sm leading-6'>
            {t('Create an account and send your first request in minutes.')}
          </p>
        </div>
        <div className='flex shrink-0 flex-wrap gap-2.5'>
          <Button
            variant='secondary'
            className='h-10 px-4'
            render={<Link to={props.isAuthenticated ? '/keys' : '/sign-up'} />}
          >
            {t(props.isAuthenticated ? 'Manage API keys' : 'Get Started')}
            <HugeiconsIcon icon={ArrowRight01Icon} size={16} strokeWidth={2} />
          </Button>
          <Button
            variant='ghost'
            className='text-background hover:bg-background/10 hover:text-background h-10 px-4'
            render={<Link to='/pricing' />}
          >
            {t('View Pricing')}
          </Button>
        </div>
      </div>
    </section>
  )
}
