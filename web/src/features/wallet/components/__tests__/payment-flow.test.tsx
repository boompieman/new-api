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
import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { createInstance } from 'i18next'
import { I18nextProvider, initReactI18next } from 'react-i18next'
import { describe, expect, test, vi } from 'vitest'

import zhTW from '@/i18n/locales/zh-TW.json'

import type { PaymentMethod, TopupInfo } from '../../types'
import { RechargeFormCard } from '../recharge-form-card'

const OEN_METHOD: PaymentMethod = {
  name: 'OEN Payment',
  type: 'oen',
  min_topup: 10,
}

const MANUAL_BANK_TRANSFER_METHOD: PaymentMethod = {
  name: 'Manual Bank Transfer',
  type: 'manual_bank_transfer',
  min_topup: 10,
}

const TOPUP_INFO: TopupInfo = {
  enable_online_topup: true,
  enable_stripe_topup: false,
  enable_oen_topup: true,
  pay_methods: [OEN_METHOD],
  min_topup: 10,
  stripe_min_topup: 10,
  oen_min_topup: 10,
  amount_options: [10, 20],
  discount: {},
  enable_redemption: true,
}

interface RenderCardOptions {
  language?: 'en' | 'zh-TW'
  topupInfo?: TopupInfo
  selectedPaymentMethod?: PaymentMethod
  onPaymentMethodSelect?: (method: PaymentMethod) => void
  onSubmitPayment?: () => void
  onSelectPreset?: (preset: { value: number; discount?: number }) => void
}

async function renderCard(options: RenderCardOptions = {}) {
  const i18n = createInstance()
  await i18n.use(initReactI18next).init({
    lng: options.language || 'en',
    fallbackLng: 'en',
    resources: {
      en: { translation: {} },
      'zh-TW': zhTW,
    },
  })

  render(
    <I18nextProvider i18n={i18n}>
      <RechargeFormCard
        topupInfo={options.topupInfo || TOPUP_INFO}
        presetAmounts={[{ value: 10 }, { value: 20 }]}
        selectedPreset={10}
        onSelectPreset={options.onSelectPreset || vi.fn()}
        topupAmount={10}
        onTopupAmountChange={vi.fn()}
        paymentAmount={350}
        calculating={false}
        selectedPaymentMethod={options.selectedPaymentMethod}
        selectedWaffoMethodIndex={null}
        onPaymentMethodSelect={options.onPaymentMethodSelect || vi.fn()}
        onSubmitPayment={options.onSubmitPayment || vi.fn()}
        paymentLoading={null}
        redemptionCode=''
        onRedemptionCodeChange={vi.fn()}
        onRedeem={vi.fn()}
        redeeming={false}
        priceRatio={35}
      />
    </I18nextProvider>
  )
}

describe('wallet payment flow', () => {
  test('preset amounts use a compact single-choice control', async () => {
    const user = userEvent.setup()
    const onSelectPreset = vi.fn()
    await renderCard({ onSelectPreset })

    const amountGroup = screen.getByRole('group', { name: 'Amount' })
    const selectedAmount = within(amountGroup).getByRole('button', {
      name: '10. You Pay 350',
    })
    expect(selectedAmount).toHaveAttribute('aria-pressed', 'true')

    await user.click(
      within(amountGroup).getByRole('button', { name: '20. You Pay 700' })
    )

    expect(onSelectPreset).toHaveBeenCalledWith({ value: 20 })
  })

  test('choosing a payment method does not submit payment', async () => {
    const user = userEvent.setup()
    const onPaymentMethodSelect = vi.fn()
    const onSubmitPayment = vi.fn()
    await renderCard({ onPaymentMethodSelect, onSubmitPayment })

    await user.click(screen.getByRole('button', { name: 'OEN Payment' }))

    expect(onPaymentMethodSelect).toHaveBeenCalledWith(OEN_METHOD)
    expect(onSubmitPayment).not.toHaveBeenCalled()
  })

  test('selected payment method exposes an explicit pay action', async () => {
    const user = userEvent.setup()
    const onSubmitPayment = vi.fn()
    await renderCard({ selectedPaymentMethod: OEN_METHOD, onSubmitPayment })

    expect(screen.getByRole('button', { name: 'OEN Payment' })).toHaveAttribute(
      'aria-pressed',
      'true'
    )

    await user.click(screen.getByRole('button', { name: 'Pay 350' }))

    expect(onSubmitPayment).toHaveBeenCalledOnce()
  })

  test('selected manual bank transfer is localized and explains when bank details appear', async () => {
    await renderCard({
      language: 'zh-TW',
      topupInfo: {
        ...TOPUP_INFO,
        pay_methods: [MANUAL_BANK_TRANSFER_METHOD],
      },
      selectedPaymentMethod: MANUAL_BANK_TRANSFER_METHOD,
    })

    expect(screen.getByRole('button', { name: '手動匯款' })).toHaveAttribute(
      'aria-pressed',
      'true'
    )
    expect(
      screen.getByText('確認付款後將顯示銀行帳戶資訊與訂單編號。')
    ).toBeVisible()
  })
})
