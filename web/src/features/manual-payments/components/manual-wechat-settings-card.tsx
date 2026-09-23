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
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { ImagePlus, QrCode, Trash2 } from 'lucide-react'
import * as React from 'react'
import { useForm, type Resolver } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
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
import { Skeleton } from '@/components/ui/skeleton'
import { Switch } from '@/components/ui/switch'
import { Textarea } from '@/components/ui/textarea'
import { TitledCard } from '@/components/ui/titled-card'
import {
  ADMIN_PERMISSION_ACTIONS,
  ADMIN_PERMISSION_RESOURCES,
  hasPermission,
} from '@/lib/admin-permissions'
import { handleServerError } from '@/lib/handle-server-error'
import { createServerError } from '@/lib/server-error-message'
import { useAuthStore } from '@/stores/auth-store'

import {
  getManualWechatPaymentSettings,
  updateManualWechatPaymentSettings,
} from '../api'
import {
  createManualWechatSettingsSchema,
  MANUAL_WECHAT_INSTRUCTIONS_MAX_LENGTH,
  type ManualWechatSettingsFormValues,
  validateManualWechatQRCodeFile,
} from '../lib/manual-wechat-settings-schema'

const emptySettings: ManualWechatSettingsFormValues = {
  enabled: false,
  qr_code_image: '',
  expires_at: 0,
  instructions: '',
}

function formatExpiryInput(timestamp: number): string {
  if (!timestamp) return ''
  const date = new Date(timestamp * 1000)
  const pad = (value: number) => String(value).padStart(2, '0')
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}`
}

function parseExpiryInput(value: string): number {
  if (!value) return 0
  const milliseconds = new Date(value).getTime()
  return Number.isFinite(milliseconds) ? Math.floor(milliseconds / 1000) : 0
}

export function ManualWechatSettingsCard() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const user = useAuthStore((state) => state.auth.user)
  const canWrite = hasPermission(
    user,
    ADMIN_PERMISSION_RESOURCES.PAYMENT,
    ADMIN_PERMISSION_ACTIONS.WRITE
  )
  const fileInputRef = React.useRef<HTMLInputElement>(null)
  const schema = React.useMemo(
    () => createManualWechatSettingsSchema(Math.floor(Date.now() / 1000)),
    []
  )
  const form = useForm<ManualWechatSettingsFormValues>({
    resolver: zodResolver(schema) as Resolver<ManualWechatSettingsFormValues>,
    defaultValues: emptySettings,
    mode: 'onChange',
  })

  const settingsQuery = useQuery({
    queryKey: ['manual-wechat-payment-settings'],
    queryFn: async () => {
      const response = await getManualWechatPaymentSettings()
      if (!response.success || !response.data) {
        throw createServerError(
          response,
          t('payment.manualWechat.settings.loadFailed')
        )
      }
      return response.data
    },
  })

  React.useEffect(() => {
    if (settingsQuery.data) form.reset(settingsQuery.data)
  }, [form, settingsQuery.data])

  const saveMutation = useMutation({
    mutationFn: updateManualWechatPaymentSettings,
    onSuccess: (response) => {
      if (!response.success || !response.data) {
        handleServerError(
          response,
          t('payment.manualWechat.settings.saveFailed')
        )
        return
      }
      form.reset(response.data)
      queryClient.setQueryData(
        ['manual-wechat-payment-settings'],
        response.data
      )
      queryClient.invalidateQueries({ queryKey: ['manual-wechat-info'] })
      toast.success(t('payment.manualWechat.settings.saved'))
    },
    onError: (error: Error) => {
      handleServerError(error, t('payment.manualWechat.settings.saveFailed'))
    },
  })

  const handleFile = async (file: File | undefined) => {
    if (!file) return
    const errorKey = validateManualWechatQRCodeFile(file)
    if (errorKey) {
      form.setError('qr_code_image', { message: errorKey })
      return
    }
    try {
      const dataUrl = await new Promise<string>((resolve, reject) => {
        const reader = new FileReader()
        reader.addEventListener('load', () => {
          if (typeof reader.result === 'string') resolve(reader.result)
          else reject(new Error('invalid file result'))
        })
        reader.addEventListener('error', () => reject(reader.error))
        reader.readAsDataURL(file)
      })
      form.clearErrors('qr_code_image')
      form.setValue('qr_code_image', dataUrl, {
        shouldDirty: true,
        shouldValidate: true,
      })
    } catch (error) {
      handleServerError(
        error,
        t('payment.manualWechat.settings.fileReadFailed')
      )
    } finally {
      if (fileInputRef.current) fileInputRef.current.value = ''
    }
  }

  const clearQRCode = () => {
    form.setValue('qr_code_image', '', {
      shouldDirty: true,
      shouldValidate: true,
    })
    form.setValue('enabled', false, {
      shouldDirty: true,
      shouldValidate: true,
    })
  }

  if (settingsQuery.isLoading) {
    return <Skeleton className='mt-8 h-80 w-full rounded-xl' />
  }

  const image = form.watch('qr_code_image')

  return (
    <TitledCard
      title={t('payment.manualWechat.settings.title')}
      description={t('payment.manualWechat.settings.description')}
      icon={<QrCode className='h-4 w-4' aria-hidden='true' />}
      disableHoverEffect
      className='mt-8'
    >
      <Form {...form}>
        <form
          className='space-y-5'
          onSubmit={form.handleSubmit((values) => {
            if (canWrite) saveMutation.mutate(values)
          })}
        >
          <FormField
            control={form.control}
            name='enabled'
            render={({ field }) => (
              <FormItem className='flex items-center justify-between gap-4 rounded-lg border p-4'>
                <div className='space-y-1'>
                  <FormLabel>
                    {t('payment.manualWechat.settings.enabled')}
                  </FormLabel>
                  <FormDescription>
                    {t('payment.manualWechat.settings.enabledDescription')}
                  </FormDescription>
                </div>
                <FormControl>
                  <Switch
                    checked={field.value}
                    onCheckedChange={field.onChange}
                    disabled={!canWrite}
                    aria-label={t('payment.manualWechat.settings.enabled')}
                  />
                </FormControl>
              </FormItem>
            )}
          />

          <FormField
            control={form.control}
            name='qr_code_image'
            render={() => (
              <FormItem>
                <FormLabel>
                  {t('payment.manualWechat.settings.qrCode')}
                </FormLabel>
                <FormControl>
                  <div className='space-y-3'>
                    <input
                      ref={fileInputRef}
                      type='file'
                      accept='image/png,image/jpeg,image/webp'
                      disabled={!canWrite}
                      className='sr-only'
                      aria-label={t('payment.manualWechat.settings.qrCodeFile')}
                      onChange={(event) =>
                        void handleFile(event.target.files?.[0])
                      }
                    />
                    {image ? (
                      <div className='flex flex-col gap-3 rounded-lg border p-3 sm:flex-row sm:items-center'>
                        <div className='flex h-44 w-full justify-center overflow-hidden rounded-md bg-white p-2 sm:w-44'>
                          <img
                            src={image}
                            alt={t(
                              'payment.manualWechat.settings.qrCodePreview'
                            )}
                            className='h-full w-full object-contain'
                          />
                        </div>
                        <div className='flex flex-1 flex-col gap-2'>
                          <Button
                            type='button'
                            variant='outline'
                            disabled={!canWrite}
                            onClick={() => fileInputRef.current?.click()}
                          >
                            <ImagePlus className='h-4 w-4' aria-hidden='true' />
                            {t('payment.manualWechat.settings.replaceQrCode')}
                          </Button>
                          <Button
                            type='button'
                            variant='destructive'
                            disabled={!canWrite}
                            onClick={clearQRCode}
                          >
                            <Trash2 className='h-4 w-4' aria-hidden='true' />
                            {t('payment.manualWechat.settings.clearQrCode')}
                          </Button>
                        </div>
                      </div>
                    ) : (
                      <Button
                        type='button'
                        variant='outline'
                        disabled={!canWrite}
                        className='w-full'
                        onClick={() => fileInputRef.current?.click()}
                      >
                        <ImagePlus className='h-4 w-4' aria-hidden='true' />
                        {t('payment.manualWechat.settings.uploadQrCode')}
                      </Button>
                    )}
                  </div>
                </FormControl>
                <FormDescription>
                  {t('payment.manualWechat.settings.fileRequirements')}
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />

          <FormField
            control={form.control}
            name='expires_at'
            render={({ field }) => (
              <FormItem>
                <FormLabel>
                  {t('payment.manualWechat.settings.expiresAt')}
                </FormLabel>
                <FormControl>
                  <Input
                    type='datetime-local'
                    disabled={!canWrite}
                    value={formatExpiryInput(field.value)}
                    onChange={(event) =>
                      field.onChange(parseExpiryInput(event.target.value))
                    }
                  />
                </FormControl>
                <FormDescription>
                  {t('payment.manualWechat.settings.expiresAtDescription')}
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
                <FormLabel>
                  {t('payment.manualWechat.settings.instructions')}
                </FormLabel>
                <FormControl>
                  <Textarea
                    {...field}
                    disabled={!canWrite}
                    maxLength={MANUAL_WECHAT_INSTRUCTIONS_MAX_LENGTH}
                    rows={4}
                    placeholder={t(
                      'payment.manualWechat.settings.instructionsPlaceholder'
                    )}
                  />
                </FormControl>
                <FormDescription>
                  {t('payment.manualWechat.settings.instructionsDescription')}
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />

          <Alert>
            <AlertDescription>
              {t('payment.manualWechat.settings.securityNotice')}
            </AlertDescription>
          </Alert>

          <div className='flex justify-end'>
            <Button
              type='submit'
              disabled={
                !canWrite || saveMutation.isPending || !form.formState.isDirty
              }
            >
              {saveMutation.isPending
                ? t('payment.manualWechat.settings.saving')
                : t('payment.manualWechat.settings.save')}
            </Button>
          </div>
        </form>
      </Form>
    </TitledCard>
  )
}
