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
import { Link } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'

import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Separator } from '@/components/ui/separator'

import { CodeExample } from './code-example'

const endpoints = [
  ['GET', '/v1/models', 'List available models'],
  ['POST', '/v1/chat/completions', 'Generate a chat completion'],
  ['POST', '/v1/responses', 'Use the Responses API'],
  ['POST', '/v1/embeddings', 'Create text embeddings'],
  ['POST', '/v1/images/generations', 'Generate an image'],
  ['POST', '/v1/audio/speech', 'Create speech audio'],
  ['POST', '/v1/audio/transcriptions', 'Transcribe audio'],
] as const

const errorRows = [
  ['400', 'The request is invalid or contains unsupported parameters.'],
  ['401', 'The API key is missing or invalid.'],
  ['403', 'The key cannot access the requested model or resource.'],
  ['429', 'The request exceeded a rate or quota limit.'],
  ['500+', 'A temporary gateway or upstream provider error occurred.'],
] as const

const capabilities = [
  [
    'Streaming responses',
    'Set stream to true to receive incremental output when the selected model and provider support streaming.',
  ],
  [
    'Tool calling',
    'Send tool definitions with supported chat or response models, then execute returned tool calls in your application.',
  ],
  [
    'Multimodal input',
    'Use supported text, image, and audio inputs with the request shape required by your chosen protocol.',
  ],
] as const

const checklistItems = [
  'Create a dedicated API key and keep it in a server-side secret store.',
  'Choose a compatible model from the live model catalog.',
  'Set the protocol-specific base URL and authentication headers.',
  'Test streaming, tool calling, and multimodal inputs before enabling them in production.',
  'Retry rate limits and temporary provider errors with exponential backoff.',
  'Monitor usage, cost, and request logs after launch.',
] as const

type ApiReferenceSectionProps = {
  apiOrigin: string
}

export function ApiReferenceSection(props: ApiReferenceSectionProps) {
  const { t } = useTranslation()

  return (
    <>
      <Separator />
      <section id='protocols' className='scroll-mt-32 py-12'>
        <h2 className='text-3xl font-semibold tracking-tight'>
          {t('API protocols')}
        </h2>
        <p className='text-muted-foreground mt-3 leading-7'>
          {t(
            'Use one API key with OpenAI, Anthropic, or Gemini request formats. Choose the protocol that best matches your existing client.'
          )}
        </p>

        <div className='mt-6 grid gap-4'>
          <Card size='sm'>
            <CardHeader>
              <CardTitle>OpenAI</CardTitle>
              <CardDescription>
                {t(
                  'Use the /v1 base path with the OpenAI SDK or any OpenAI-compatible client.'
                )}
              </CardDescription>
            </CardHeader>
            <CardContent>
              <CodeExample
                code={`Base URL: ${props.apiOrigin}/v1\nAuthorization: Bearer $OMNIAI_API_KEY`}
                label='OpenAI'
              />
            </CardContent>
          </Card>

          <Card size='sm'>
            <CardHeader>
              <CardTitle>Anthropic</CardTitle>
              <CardDescription>
                {t(
                  'Send native Messages API requests with the Anthropic authentication and version headers.'
                )}
              </CardDescription>
            </CardHeader>
            <CardContent>
              <CodeExample
                code={`POST ${props.apiOrigin}/v1/messages\nx-api-key: $OMNIAI_API_KEY\nanthropic-version: 2023-06-01`}
                label='Anthropic'
              />
            </CardContent>
          </Card>

          <Card size='sm'>
            <CardHeader>
              <CardTitle>Gemini</CardTitle>
              <CardDescription>
                {t(
                  'Send native generateContent requests with the model name in the URL.'
                )}
              </CardDescription>
            </CardHeader>
            <CardContent>
              <CodeExample
                code={`POST ${props.apiOrigin}/v1beta/models/{model}:generateContent\nx-goog-api-key: $OMNIAI_API_KEY`}
                label='Gemini'
              />
            </CardContent>
          </Card>
        </div>
      </section>

      <Separator />
      <section id='capabilities' className='scroll-mt-32 py-12'>
        <h2 className='text-3xl font-semibold tracking-tight'>
          {t('Core capabilities')}
        </h2>
        <p className='text-muted-foreground mt-3 leading-7'>
          {t(
            'Build richer AI workflows while keeping protocol and model compatibility in mind.'
          )}
        </p>
        <div className='mt-6 grid gap-4 sm:grid-cols-3'>
          {capabilities.map((capability) => (
            <Card key={capability[0]} size='sm'>
              <CardHeader>
                <CardTitle>{t(capability[0])}</CardTitle>
                <CardDescription>{t(capability[1])}</CardDescription>
              </CardHeader>
            </Card>
          ))}
        </div>
        <Alert className='mt-6'>
          <AlertTitle>{t('Compatibility varies by model')}</AlertTitle>
          <AlertDescription>
            {t(
              'Available features depend on the selected model and upstream provider.'
            )}{' '}
            <Link
              to='/pricing'
              className='font-medium underline underline-offset-4'
            >
              {t('Check the live model catalog')}
            </Link>
            .
          </AlertDescription>
        </Alert>
      </section>

      <Separator />
      <section id='endpoints' className='scroll-mt-32 py-12'>
        <h2 className='text-3xl font-semibold tracking-tight'>
          {t('API endpoints')}
        </h2>
        <p className='text-muted-foreground mt-3 leading-7'>
          {t(
            'OpenAI-compatible routes cover the most common text, multimodal, image, and audio workflows.'
          )}
        </p>

        <div className='border-border mt-6 overflow-hidden rounded-xl border'>
          <div className='bg-muted/40 text-muted-foreground grid grid-cols-[5.5rem_minmax(0,1fr)] gap-3 border-b px-4 py-3 text-xs font-semibold tracking-wider uppercase sm:grid-cols-[5.5rem_15rem_minmax(0,1fr)]'>
            <span>{t('Method')}</span>
            <span>{t('Endpoint')}</span>
            <span className='hidden sm:block'>{t('Purpose')}</span>
          </div>
          {endpoints.map((endpoint) => (
            <div
              key={endpoint[1]}
              className='border-border/70 grid grid-cols-[5.5rem_minmax(0,1fr)] gap-3 border-b px-4 py-3 text-sm last:border-b-0 sm:grid-cols-[5.5rem_15rem_minmax(0,1fr)]'
            >
              <Badge variant={endpoint[0] === 'GET' ? 'secondary' : 'outline'}>
                {endpoint[0]}
              </Badge>
              <code className='min-w-0 font-mono text-xs break-all sm:text-sm'>
                {endpoint[1]}
              </code>
              <span className='text-muted-foreground col-start-2 sm:col-start-auto'>
                {t(endpoint[2])}
              </span>
            </div>
          ))}
        </div>
      </section>

      <Separator />
      <section id='errors' className='scroll-mt-32 py-12'>
        <h2 className='text-3xl font-semibold tracking-tight'>
          {t('Error handling')}
        </h2>
        <p className='text-muted-foreground mt-3 leading-7'>
          {t(
            'Errors use standard HTTP status codes and return a JSON body with a readable message.'
          )}
        </p>
        <div className='mt-6 flex flex-col gap-3'>
          {errorRows.map((row) => (
            <div
              key={row[0]}
              className='bg-muted/35 grid grid-cols-[4.5rem_minmax(0,1fr)] gap-3 rounded-lg px-4 py-3 text-sm'
            >
              <code className='font-mono font-semibold'>{row[0]}</code>
              <span className='text-muted-foreground'>{t(row[1])}</span>
            </div>
          ))}
        </div>
        <p className='text-muted-foreground mt-6 text-sm leading-6'>
          {t(
            'For 429 and 500-level responses, retry with exponential backoff and log the request ID for support.'
          )}
        </p>
      </section>

      <Separator />
      <section id='checklist' className='scroll-mt-32 py-12'>
        <Badge variant='outline'>{t('Before production')}</Badge>
        <h2 className='mt-4 text-3xl font-semibold tracking-tight'>
          {t('Integration checklist')}
        </h2>
        <p className='text-muted-foreground mt-3 leading-7'>
          {t(
            'Use this checklist to move from a successful first request to a reliable production integration.'
          )}
        </p>
        <ol className='mt-6 flex flex-col gap-3'>
          {checklistItems.map((item, index) => (
            <li
              key={item}
              className='bg-muted/35 grid grid-cols-[2rem_minmax(0,1fr)] items-start gap-3 rounded-lg px-4 py-3 text-sm'
            >
              <span className='bg-foreground text-background flex size-6 items-center justify-center rounded-full text-xs font-semibold'>
                {index + 1}
              </span>
              <span className='pt-0.5 leading-6'>{t(item)}</span>
            </li>
          ))}
        </ol>
      </section>
    </>
  )
}
