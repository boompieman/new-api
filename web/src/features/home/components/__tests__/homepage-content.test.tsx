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
import { render, screen } from '@testing-library/react'
import type { AnchorHTMLAttributes, ReactNode } from 'react'
import { describe, expect, test, vi } from 'vitest'

import { FeaturedModels } from '../sections/featured-models'
import { Stats } from '../sections/stats'

vi.mock('@lobehub/icons', () => {
  const Icon = () => <span aria-hidden='true' />
  return {
    Claude: Icon,
    DeepSeek: Icon,
    Gemini: Icon,
    Grok: Icon,
    Meta: Icon,
    Mistral: Icon,
    OpenAI: Icon,
    Qwen: Icon,
  }
})

vi.mock('@tanstack/react-router', () => ({
  Link: (
    props: AnchorHTMLAttributes<HTMLAnchorElement> & {
      children?: ReactNode
      params?: { modelId?: string }
      to: string
    }
  ) => {
    const { to, params, children, ...anchorProps } = props
    const href = params?.modelId
      ? to.replace('$modelId', encodeURIComponent(params.modelId))
      : to
    return (
      <a href={href} {...anchorProps}>
        {children}
      </a>
    )
  },
}))

vi.mock('react-i18next', () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}))

vi.mock('@/features/pricing/hooks/use-pricing-data', () => ({
  usePricingData: () => ({
    models: [
      {
        id: 1,
        model_name: 'misc-model',
        vendor_name: 'Other',
        quota_type: 0,
        model_ratio: 1,
        completion_ratio: 1,
        enable_groups: [],
      },
      {
        id: 2,
        model_name: 'gpt-4.1',
        vendor_name: 'OpenAI',
        quota_type: 0,
        model_ratio: 1,
        completion_ratio: 1,
        enable_groups: [],
      },
      {
        id: 3,
        model_name: 'claude-sonnet-4-20250514',
        vendor_name: 'Claude',
        quota_type: 0,
        model_ratio: 1,
        completion_ratio: 1,
        enable_groups: [],
      },
      {
        id: 4,
        model_name: 'gemini-2.5-pro',
        vendor_name: 'Gemini',
        quota_type: 0,
        model_ratio: 1,
        completion_ratio: 1,
        enable_groups: [],
      },
      {
        id: 5,
        model_name: 'deepseek-chat',
        vendor_name: 'DeepSeek',
        quota_type: 0,
        model_ratio: 1,
        completion_ratio: 1,
        enable_groups: [],
      },
      {
        id: 6,
        model_name: 'gpt-5.6-sol',
        vendor_name: 'OpenAI',
        quota_type: 0,
        model_ratio: 1,
        completion_ratio: 1,
        enable_groups: [],
      },
      {
        id: 7,
        model_name: 'claude-fable-5',
        vendor_name: 'Claude',
        quota_type: 0,
        model_ratio: 1,
        completion_ratio: 1,
        enable_groups: [],
      },
      {
        id: 8,
        model_name: 'gemini-3.1-pro-preview',
        vendor_name: 'Gemini',
        quota_type: 0,
        model_ratio: 1,
        completion_ratio: 1,
        enable_groups: [],
      },
      {
        id: 9,
        model_name: 'deepseek-v4-pro',
        vendor_name: 'DeepSeek',
        quota_type: 0,
        model_ratio: 1,
        completion_ratio: 1,
        enable_groups: [],
      },
    ],
    isLoading: false,
    priceRate: 1,
    usdExchangeRate: 1,
  }),
}))

vi.mock('@/features/pricing/lib/price', () => ({
  formatPrice: () => '$1',
  formatRequestPrice: () => '$1',
}))

describe('homepage product content', () => {
  test('provider strip uses four columns on mobile and eight on desktop', () => {
    render(<Stats />)

    const providerLabel = screen.getByText('OpenAI')
    const providerGrid = providerLabel.parentElement?.parentElement

    expect(
      screen.getByRole('region', { name: 'Supported AI providers' })
    ).toBeInTheDocument()
    expect(providerGrid).toHaveClass('grid-cols-4', 'md:grid-cols-8')
  })

  test('featured models prefer the newest configured model from each major provider', () => {
    render(<FeaturedModels />)

    for (const modelName of [
      'gpt-5.6-sol',
      'claude-fable-5',
      'gemini-3.1-pro-preview',
      'deepseek-v4-pro',
    ]) {
      expect(screen.getByText(modelName).closest('a')).toHaveAttribute(
        'href',
        `/pricing/${modelName}`
      )
    }
    expect(screen.queryByText('misc-model')).not.toBeInTheDocument()
  })
})
