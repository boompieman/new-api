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
import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { usePricingData } from '@/features/pricing/hooks/use-pricing-data'
import { isTokenBasedModel } from '@/features/pricing/lib/model-helpers'
import { formatPrice, formatRequestPrice } from '@/features/pricing/lib/price'

const FEATURED_PROVIDERS = [
  {
    hint: 'openai',
    preferredModels: [
      'gpt-5.6-sol',
      'gpt-5.6',
      'gpt-5.6-terra',
      'gpt-5.5',
      'gpt-5.4',
    ],
  },
  {
    hint: 'claude',
    preferredModels: [
      'claude-fable-5',
      'claude-opus-5',
      'claude-sonnet-5',
      'claude-opus-4-8',
      'claude-opus-4-7',
      'claude-opus-4-6',
      'claude-sonnet-4-6',
      'claude-sonnet-4-5-20250929',
      'claude-sonnet-4-20250514',
    ],
  },
  {
    hint: 'gemini',
    preferredModels: [
      'gemini-3.1-pro-preview',
      'gemini-3-pro-preview',
      'gemini-2.5-pro',
    ],
  },
  {
    hint: 'deepseek',
    preferredModels: [
      'deepseek-v4-pro',
      'deepseek-v4-flash',
      'deepseek-reasoner',
      'deepseek-chat',
    ],
  },
] as const

export function FeaturedModels() {
  const { t } = useTranslation()
  const { models, isLoading, priceRate, usdExchangeRate } = usePricingData()
  const featuredModels = useMemo(() => {
    const selected = []
    const selectedNames = new Set<string>()

    for (const provider of FEATURED_PROVIDERS) {
      const providerModels = models.filter((candidate) => {
        const searchable =
          `${candidate.vendor_name || ''} ${candidate.model_name}`
            .toLowerCase()
            .trim()
        return searchable.includes(provider.hint)
      })

      const model =
        provider.preferredModels
          .map((modelName) =>
            providerModels.find(
              (candidate) => candidate.model_name.toLowerCase() === modelName
            )
          )
          .find((candidate) => candidate !== undefined) || providerModels[0]

      if (model) {
        selected.push(model)
        selectedNames.add(model.model_name)
      }
    }

    for (const model of models) {
      if (selected.length >= 4) break
      if (!selectedNames.has(model.model_name)) {
        selected.push(model)
        selectedNames.add(model.model_name)
      }
    }

    return selected
  }, [models])

  return (
    <section className='border-b px-5 py-16 sm:px-6 md:py-20'>
      <div className='mx-auto max-w-6xl'>
        <div className='mb-8 flex flex-col justify-between gap-5 sm:flex-row sm:items-end'>
          <div>
            <p className='text-muted-foreground mb-2 text-xs font-medium tracking-[0.14em] uppercase'>
              {t('Popular models')}
            </p>
            <h2 className='text-2xl font-semibold tracking-tight md:text-3xl'>
              {t('Connect the models your team already uses')}
            </h2>
            <p className='text-muted-foreground mt-3 max-w-2xl text-sm leading-6'>
              {t(
                'Browse live model availability and pricing from your configured upstream providers.'
              )}
            </p>
          </div>
          <Button
            variant='ghost'
            className='text-muted-foreground self-start px-0 sm:self-auto'
            render={<Link to='/pricing' />}
          >
            {t('View all models')}
            <HugeiconsIcon icon={ArrowRight01Icon} size={15} strokeWidth={2} />
          </Button>
        </div>

        {isLoading && (
          <div className='grid gap-3 sm:grid-cols-2 lg:grid-cols-4'>
            {Array.from({ length: 4 }, (_, index) => (
              <Skeleton key={index} className='h-40 rounded-lg' />
            ))}
          </div>
        )}
        {!isLoading && featuredModels.length > 0 && (
          <div className='grid gap-3 sm:grid-cols-2 lg:grid-cols-4'>
            {featuredModels.map((model) => {
              const tokenBased = isTokenBasedModel(model)
              const dynamicPricing = model.billing_mode === 'tiered_expr'
              return (
                <Link
                  key={model.model_name}
                  to='/pricing/$modelId'
                  params={{ modelId: model.model_name }}
                  className='group bg-background hover:bg-muted/30 flex min-h-40 flex-col rounded-lg border p-5 transition-colors'
                >
                  <div className='flex items-center gap-3'>
                    <span className='bg-foreground text-background flex size-9 shrink-0 items-center justify-center rounded-lg text-sm font-semibold'>
                      {model.model_name.charAt(0).toUpperCase()}
                    </span>
                    <div className='min-w-0'>
                      <p className='truncate font-mono text-sm font-semibold'>
                        {model.model_name}
                      </p>
                      <p className='text-muted-foreground truncate text-xs'>
                        {model.vendor_name || t('Model')}
                      </p>
                    </div>
                  </div>

                  <div className='mt-auto pt-6'>
                    {dynamicPricing && (
                      <p className='text-sm font-medium'>
                        {t('Dynamic Pricing')}
                      </p>
                    )}
                    {!dynamicPricing && tokenBased && (
                      <div className='grid grid-cols-2 gap-3 text-xs'>
                        <div>
                          <p className='text-muted-foreground'>{t('Input')}</p>
                          <p className='mt-1 font-mono font-semibold'>
                            {formatPrice(
                              model,
                              'input',
                              'M',
                              false,
                              priceRate,
                              usdExchangeRate
                            )}
                          </p>
                        </div>
                        <div>
                          <p className='text-muted-foreground'>{t('Output')}</p>
                          <p className='mt-1 font-mono font-semibold'>
                            {formatPrice(
                              model,
                              'output',
                              'M',
                              false,
                              priceRate,
                              usdExchangeRate
                            )}
                          </p>
                        </div>
                      </div>
                    )}
                    {!dynamicPricing && !tokenBased && (
                      <p className='font-mono text-sm font-semibold'>
                        {formatRequestPrice(
                          model,
                          false,
                          priceRate,
                          usdExchangeRate
                        )}{' '}
                        <span className='text-muted-foreground font-sans text-xs font-normal'>
                          / {t('request')}
                        </span>
                      </p>
                    )}
                  </div>
                </Link>
              )
            })}
          </div>
        )}
        {!isLoading && featuredModels.length === 0 && (
          <div className='text-muted-foreground rounded-lg border border-dashed px-6 py-12 text-center text-sm'>
            {t('No models available')}
          </div>
        )}

        <p className='text-muted-foreground mt-4 text-xs'>
          {t('Pricing updates automatically from your configured models.')}
        </p>
      </div>
    </section>
  )
}
