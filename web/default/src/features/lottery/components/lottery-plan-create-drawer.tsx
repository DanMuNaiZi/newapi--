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
import { useTranslation } from 'react-i18next'

import {
  sideDrawerContentClassName,
  sideDrawerHeaderClassName,
} from '@/components/drawer-layout'
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet'

import { LotteryPlanForm } from './lottery-plan-form'

type LotteryPlanCreateDrawerProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function LotteryPlanCreateDrawer(props: LotteryPlanCreateDrawerProps) {
  const { t } = useTranslation()

  return (
    <Sheet open={props.open} onOpenChange={props.onOpenChange}>
      <SheetContent className={sideDrawerContentClassName('sm:max-w-[900px]')}>
        <SheetHeader className={sideDrawerHeaderClassName()}>
          <SheetTitle>{t('Create lottery plan')}</SheetTitle>
          <SheetDescription>
            {t(
              'Configure the schedule, participation eligibility, and prizes for this lottery.'
            )}
          </SheetDescription>
        </SheetHeader>
        <LotteryPlanForm
          drawer
          formId='lottery-plan-create-form'
          onCancel={() => props.onOpenChange(false)}
          onCreated={() => props.onOpenChange(false)}
        />
      </SheetContent>
    </Sheet>
  )
}
