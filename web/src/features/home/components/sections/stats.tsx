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
  Claude,
  DeepSeek,
  Gemini,
  Grok,
  Meta,
  Mistral,
  OpenAI,
  Qwen,
} from '@lobehub/icons'
import { useTranslation } from 'react-i18next'

const PROVIDERS = [
  { name: 'OpenAI', icon: <OpenAI size={24} /> },
  { name: 'Claude', icon: <Claude size={24} /> },
  { name: 'Gemini', icon: <Gemini size={24} /> },
  { name: 'Grok', icon: <Grok size={24} /> },
  { name: 'Meta', icon: <Meta size={24} /> },
  { name: 'Qwen', icon: <Qwen size={24} /> },
  { name: 'DeepSeek', icon: <DeepSeek size={24} /> },
  { name: 'Mistral', icon: <Mistral size={24} /> },
] as const

interface StatsProps {
  className?: string
}

export function Stats(_props: StatsProps) {
  const { t } = useTranslation()

  return (
    <section
      className='border-b px-5 py-9 sm:px-6 md:py-10'
      aria-label={t('Supported AI providers')}
    >
      <div className='mx-auto max-w-6xl'>
        <p className='text-muted-foreground mb-6 text-center text-xs font-medium tracking-[0.14em] uppercase'>
          {t('Supported AI providers')}
        </p>
        <div className='grid grid-cols-4 gap-x-4 gap-y-7 md:grid-cols-8'>
          {PROVIDERS.map((provider) => (
            <div
              key={provider.name}
              className='text-muted-foreground hover:text-foreground flex flex-col items-center gap-2 text-xs grayscale transition-all duration-200 hover:grayscale-0'
            >
              {provider.icon}
              <span>{provider.name}</span>
            </div>
          ))}
        </div>
      </div>
    </section>
  )
}
