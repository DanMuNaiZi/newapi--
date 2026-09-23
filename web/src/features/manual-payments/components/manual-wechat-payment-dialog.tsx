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
import { CircleAlert, Download, QrCode, Smartphone } from 'lucide-react'
import * as React from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { CopyButton } from '@/components/copy-button'
import { Dialog } from '@/components/dialog'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Separator } from '@/components/ui/separator'
import { toIntlLocale } from '@/i18n/languages'

import { isManualWechatQRCodeExpired } from '../lib/manual-wechat-order'
import type { ManualWechatOrder } from '../types'

interface ManualWechatPaymentDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  order: ManualWechatOrder | null
}

function qrCodeExtension(dataUrl: string): string {
  if (dataUrl.startsWith('data:image/jpeg;')) return 'jpg'
  if (dataUrl.startsWith('data:image/webp;')) return 'webp'
  return 'png'
}

export function ManualWechatPaymentDialog(
  props: ManualWechatPaymentDialogProps
) {
  const { t, i18n } = useTranslation()
  const [nowSeconds, setNowSeconds] = React.useState(() =>
    Math.floor(Date.now() / 1000)
  )

  React.useEffect(() => {
    if (!props.open) return undefined
    setNowSeconds(Math.floor(Date.now() / 1000))
    const timer = window.setInterval(
      () => setNowSeconds(Math.floor(Date.now() / 1000)),
      1_000
    )
    return () => window.clearInterval(timer)
  }, [props.open])

  if (!props.order) return null
  const order = props.order

  const expired = isManualWechatQRCodeExpired(
    order.qr_code_expires_at,
    nowSeconds
  )
  const locale = toIntlLocale(i18n.resolvedLanguage || i18n.language)
  let purpose = '-'
  let purposeLabel = t('payment.manualWechat.plan')
  if (order.kind === 'topup') {
    purposeLabel = t('payment.manualWechat.topupAmount')
    if ('topup_amount' in order.purpose) {
      purpose = String(order.purpose.topup_amount)
    }
  } else if ('plan_title' in order.purpose) {
    purpose = order.purpose.plan_title
  }

  const saveQRCode = () => {
    const link = document.createElement('a')
    link.href = order.qr_code_image
    link.download = `${order.trade_no}.${qrCodeExtension(link.href)}`
    link.click()
  }

  const finish = () => {
    toast.success(t('payment.manualWechat.waitingForConfirmation'))
    props.onOpenChange(false)
  }

  return (
    <Dialog
      open={props.open}
      onOpenChange={props.onOpenChange}
      title={
        <span className='flex items-center gap-2'>
          <QrCode className='h-5 w-5' aria-hidden='true' />
          {t('payment.manualWechat.title')}
        </span>
      }
      description={t('payment.manualWechat.remarkRequired')}
      contentClassName='max-sm:w-[calc(100vw-1.5rem)] sm:max-w-lg'
      bodyClassName='space-y-4'
      footer={
        <Button onClick={finish} disabled={expired}>
          {t('payment.manualWechat.completedPayment')}
        </Button>
      }
    >
      {expired ? (
        <Alert variant='destructive'>
          <CircleAlert className='h-4 w-4' aria-hidden='true' />
          <AlertTitle>{t('payment.manualWechat.qrExpired')}</AlertTitle>
          <AlertDescription>
            {t('payment.manualWechat.qrExpiredDescription')}
          </AlertDescription>
        </Alert>
      ) : null}

      <div className='flex justify-center rounded-lg border bg-white p-3 sm:p-4'>
        <img
          src={props.order.qr_code_image}
          alt={t('payment.manualWechat.qrCodeAlt')}
          className='max-h-[22rem] w-full max-w-sm object-contain'
        />
      </div>

      <div className='grid grid-cols-2 gap-3 rounded-lg border p-3 text-sm'>
        <span className='text-muted-foreground'>
          {t('payment.manualWechat.amountDue')}
        </span>
        <strong className='text-right text-lg'>
          ¥{props.order.amount_cny.toFixed(2)} CNY
        </strong>
        <span className='text-muted-foreground'>
          {t('payment.manualWechat.userId')}
        </span>
        <span className='text-right font-medium'>{props.order.user_id}</span>
        <span className='text-muted-foreground'>{purposeLabel}</span>
        <span className='text-right font-medium'>{purpose}</span>
        <span className='text-muted-foreground'>
          {t('payment.manualWechat.orderNumber')}
        </span>
        <span className='flex min-w-0 items-center justify-end gap-1'>
          <span className='truncate font-mono text-xs font-semibold'>
            {props.order.trade_no}
          </span>
          <CopyButton
            value={props.order.trade_no}
            size='icon'
            tooltip={t('payment.manualWechat.copyOrderNumber')}
          />
        </span>
        <span className='text-muted-foreground'>
          {t('payment.manualWechat.qrExpiresAt')}
        </span>
        <span className='text-right text-xs'>
          {props.order.qr_code_expires_at > 0
            ? new Date(props.order.qr_code_expires_at * 1000).toLocaleString(
                locale
              )
            : t('payment.manualWechat.neverExpires')}
        </span>
      </div>

      <Alert variant={expired ? 'destructive' : 'default'}>
        <AlertTitle>{t('payment.manualWechat.remarkRequiredTitle')}</AlertTitle>
        <AlertDescription className='space-y-2'>
          <p>{props.order.instructions}</p>
          <Badge variant='outline' className='font-mono'>
            {props.order.trade_no}
          </Badge>
        </AlertDescription>
      </Alert>

      <div className='flex flex-col gap-2 sm:flex-row'>
        <CopyButton
          value={props.order.trade_no}
          variant='outline'
          size='default'
          className='flex-1'
          tooltip={t('payment.manualWechat.copyOrderNumber')}
        >
          {t('payment.manualWechat.copyOrderNumber')}
        </CopyButton>
        <Button
          type='button'
          variant='outline'
          className='flex-1'
          onClick={saveQRCode}
          disabled={expired}
        >
          <Download className='h-4 w-4' aria-hidden='true' />
          {t('payment.manualWechat.saveQrCode')}
        </Button>
      </div>

      <Separator />
      <p className='text-muted-foreground flex items-start gap-2 text-xs sm:hidden'>
        <Smartphone className='mt-0.5 h-4 w-4 shrink-0' aria-hidden='true' />
        {t('payment.manualWechat.mobileHint')}
      </p>
      <p className='text-muted-foreground text-xs'>
        {t('payment.manualWechat.manualConfirmationNotice')}
      </p>
    </Dialog>
  )
}
