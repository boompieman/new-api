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
import { InformationCircleIcon } from '@hugeicons/core-free-icons'
import { HugeiconsIcon } from '@hugeicons/react'
import { useTranslation } from 'react-i18next'

import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Separator } from '@/components/ui/separator'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'

import { CodeExample } from './code-example'

type QuickStartSectionProps = {
  apiOrigin: string
}

export function QuickStartSection(props: QuickStartSectionProps) {
  const { t } = useTranslation()
  const pythonCode = `# pip install openai
import os
from openai import OpenAI

client = OpenAI(
    base_url="${props.apiOrigin}/v1",
    api_key=os.environ["OMNIAI_API_KEY"],
)

response = client.chat.completions.create(
    model="gpt-4o-mini",
    messages=[{"role": "user", "content": "Hello, omniAI!"}],
)

print(response.choices[0].message.content)`
  const typescriptCode = `// npm install openai
import OpenAI from "openai";

const client = new OpenAI({
  baseURL: "${props.apiOrigin}/v1",
  apiKey: process.env.OMNIAI_API_KEY,
});

const response = await client.chat.completions.create({
  model: "gpt-4o-mini",
  messages: [{ role: "user", content: "Hello, omniAI!" }],
});

console.log(response.choices[0].message.content);`
  const curlCode = `curl "${props.apiOrigin}/v1/chat/completions" \\
  -H "Authorization: Bearer $OMNIAI_API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "gpt-4o-mini",
    "messages": [{"role": "user", "content": "Hello, omniAI!"}]
  }'`

  return (
    <>
      <section id='quick-start' className='scroll-mt-32 py-12'>
        <Badge variant='outline'>{t('5 minute setup')}</Badge>
        <h2 className='mt-4 text-3xl font-semibold tracking-tight'>
          {t('Quick start')}
        </h2>
        <p className='text-muted-foreground mt-3 max-w-2xl text-base leading-7'>
          {t(
            'Use the OpenAI SDK you already know. Change the base URL and API key, then keep the rest of your integration.'
          )}
        </p>

        <div className='mt-8'>
          <h3 className='text-lg font-semibold'>
            {t('Send your first request')}
          </h3>
          <p className='text-muted-foreground mt-1 text-sm'>
            {t(
              'Choose your preferred client and replace the example model with one enabled for your API key.'
            )}
          </p>
          <Tabs defaultValue='python' className='mt-4'>
            <TabsList aria-label={t('Integration language')}>
              <TabsTrigger value='python'>Python</TabsTrigger>
              <TabsTrigger value='typescript'>TypeScript</TabsTrigger>
              <TabsTrigger value='curl'>cURL</TabsTrigger>
            </TabsList>
            <TabsContent value='python'>
              <CodeExample code={pythonCode} label='Python' />
            </TabsContent>
            <TabsContent value='typescript'>
              <CodeExample code={typescriptCode} label='TypeScript' />
            </TabsContent>
            <TabsContent value='curl'>
              <CodeExample code={curlCode} label='cURL' />
            </TabsContent>
          </Tabs>
        </div>
      </section>

      <Separator />

      <section id='authentication' className='scroll-mt-32 py-12'>
        <h2 className='text-3xl font-semibold tracking-tight'>
          {t('Authentication')}
        </h2>
        <p className='text-muted-foreground mt-3 leading-7'>
          {t(
            'Send your API key as a Bearer token in the Authorization header for every request.'
          )}
        </p>
        <CodeExample
          className='mt-6'
          code='Authorization: Bearer $OMNIAI_API_KEY'
          label='HTTP header'
        />
        <Alert className='mt-6'>
          <HugeiconsIcon icon={InformationCircleIcon} />
          <AlertTitle>{t('Keep API keys server-side')}</AlertTitle>
          <AlertDescription>
            {t(
              'Never expose a secret key in browser code, mobile apps, public repositories, or screenshots.'
            )}
          </AlertDescription>
        </Alert>
      </section>
    </>
  )
}
