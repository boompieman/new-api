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
import { zodResolver } from '@hookform/resolvers/zod'
import { useMemo } from 'react'
import { useForm, type Resolver } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { z } from 'zod'

import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { Switch } from '@/components/ui/switch'
import { Textarea } from '@/components/ui/textarea'

import {
  SettingsForm,
  SettingsFormGrid,
  SettingsSwitchContent,
  SettingsSwitchItem,
} from '../components/settings-form-layout'
import { SettingsPageFormActions } from '../components/settings-page-context'
import { SettingsSection } from '../components/settings-section'
import { useUpdateOption } from '../hooks/use-update-option'

const baseSchema = z.object({
  enabled: z.boolean(),
  bank_name: z.string().max(200),
  bank_code: z.string().max(50),
  branch_name: z.string().max(200),
  account_name: z.string().max(200),
  account_number: z.string().max(200),
  instructions: z.string().max(1000),
})

function createSchema(requiredMessage: string) {
  return baseSchema.superRefine((values, ctx) => {
    if (!values.enabled) return
    for (const field of [
      'bank_name',
      'account_name',
      'account_number',
    ] as const) {
      if (!values[field].trim()) {
        ctx.addIssue({
          code: 'custom',
          path: [field],
          message: requiredMessage,
        })
      }
    }
  })
}

type Values = z.infer<typeof baseSchema>

const EMPTY_CONFIG: Values = {
  enabled: false,
  bank_name: '',
  bank_code: '',
  branch_name: '',
  account_name: '',
  account_number: '',
  instructions: '',
}

function parseConfig(rawConfig: string): Values {
  try {
    const result = baseSchema.safeParse(JSON.parse(rawConfig))
    return result.success ? result.data : EMPTY_CONFIG
  } catch {
    return EMPTY_CONFIG
  }
}

export function ManualBankTransferSettingsSection({
  defaultConfig,
  complianceConfirmed,
}: {
  defaultConfig: string
  complianceConfirmed: boolean
}) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()
  const parsedDefault = parseConfig(defaultConfig)
  const validationSchema = useMemo(
    () => createSchema(t('Required when manual bank transfer is enabled')),
    [t]
  )
  const form = useForm<Values>({
    resolver: zodResolver(validationSchema) as unknown as Resolver<Values>,
    defaultValues: parsedDefault,
  })
  const { isDirty, isSubmitting } = form.formState

  async function onSubmit(values: Values) {
    const sanitized: Values = {
      ...values,
      bank_name: values.bank_name.trim(),
      bank_code: values.bank_code.trim(),
      branch_name: values.branch_name.trim(),
      account_name: values.account_name.trim(),
      account_number: values.account_number.trim(),
      instructions: values.instructions.trim(),
    }

    if (JSON.stringify(sanitized) === JSON.stringify(parsedDefault)) {
      toast.info(t('No changes to save'))
      return
    }

    const response = await updateOption.mutateAsync({
      key: 'manual_bank_transfer.config',
      value: JSON.stringify(sanitized),
    })
    if (response.success) {
      form.reset(sanitized)
    }
  }

  return (
    <SettingsSection title={t('Manual Bank Transfer')}>
      <Form {...form}>
        <SettingsForm onSubmit={form.handleSubmit(onSubmit)} autoComplete='off'>
          <SettingsPageFormActions
            onSave={form.handleSubmit(onSubmit)}
            isSaving={updateOption.isPending || isSubmitting}
            isSaveDisabled={!isDirty}
            saveLabel='Save manual bank transfer settings'
          />

          {!complianceConfirmed && (
            <Alert>
              <AlertTitle>
                {t('Payment compliance confirmation required')}
              </AlertTitle>
              <AlertDescription>
                {t(
                  'Complete the payment compliance confirmation before this payment method can appear in the wallet.'
                )}
              </AlertDescription>
            </Alert>
          )}

          <FormField
            control={form.control}
            name='enabled'
            render={({ field }) => (
              <SettingsSwitchItem>
                <SettingsSwitchContent>
                  <FormLabel>{t('Enable manual bank transfer')}</FormLabel>
                  <FormDescription>
                    {t(
                      'Show bank transfer as a payment method after all required bank details are saved.'
                    )}
                  </FormDescription>
                </SettingsSwitchContent>
                <FormControl>
                  <Switch
                    checked={field.value}
                    onCheckedChange={field.onChange}
                    disabled={updateOption.isPending || isSubmitting}
                  />
                </FormControl>
              </SettingsSwitchItem>
            )}
          />

          <SettingsFormGrid>
            <FormField
              control={form.control}
              name='bank_name'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Bank name')}</FormLabel>
                  <FormControl>
                    <Input
                      placeholder={t('Enter bank name')}
                      aria-invalid={Boolean(form.formState.errors.bank_name)}
                      {...field}
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name='bank_code'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Bank code')}</FormLabel>
                  <FormControl>
                    <Input placeholder={t('Optional bank code')} {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name='branch_name'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Branch name')}</FormLabel>
                  <FormControl>
                    <Input placeholder={t('Optional branch name')} {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name='account_name'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Account name')}</FormLabel>
                  <FormControl>
                    <Input
                      placeholder={t('Enter account name')}
                      aria-invalid={Boolean(form.formState.errors.account_name)}
                      {...field}
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name='account_number'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Account number')}</FormLabel>
                  <FormControl>
                    <Input
                      inputMode='numeric'
                      placeholder={t('Enter account number')}
                      aria-invalid={Boolean(
                        form.formState.errors.account_number
                      )}
                      {...field}
                    />
                  </FormControl>
                  <FormDescription>
                    {t(
                      'Shown only to signed-in users who create a transfer order.'
                    )}
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name='instructions'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Transfer instructions')}</FormLabel>
                  <FormControl>
                    <Textarea
                      placeholder={t(
                        'For example: Contact the administrator after completing the transfer.'
                      )}
                      rows={4}
                      {...field}
                    />
                  </FormControl>
                  <FormDescription>
                    {t(
                      'Optional instructions shown with the bank account details.'
                    )}
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
          </SettingsFormGrid>
        </SettingsForm>
      </Form>
    </SettingsSection>
  )
}
