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
import { Eye, Pencil, Play, Users, XCircle } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { DataTableRowActionMenu } from '@/components/data-table'
import { Button } from '@/components/ui/button'
import {
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuShortcut,
} from '@/components/ui/dropdown-menu'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'

import type { LotteryPlan } from '../types'
import type { LotteryDetailsTab } from './lottery-plan-details-drawer'

type LotteryPlanRowActionsProps = {
  cancelPending: boolean
  drawPending: boolean
  plan: LotteryPlan
  onCancel: (plan: LotteryPlan) => void
  onDraw: (plan: LotteryPlan) => void
  onEdit: (plan: LotteryPlan) => void
  onView: (plan: LotteryPlan, tab: LotteryDetailsTab) => void
}

export function LotteryPlanRowActions(props: LotteryPlanRowActionsProps) {
  const { t } = useTranslation()
  const canManage =
    props.plan.status === 'scheduled' || props.plan.status === 'open'

  return (
    <div className='-ml-1.5 flex items-center gap-1'>
      <Tooltip>
        <TooltipTrigger
          render={
            <Button
              variant='ghost'
              size='icon-sm'
              onClick={() => props.onView(props.plan, 'overview')}
              aria-label={t('View lottery details')}
            />
          }
        >
          <Eye />
        </TooltipTrigger>
        <TooltipContent>{t('View lottery details')}</TooltipContent>
      </Tooltip>
      <Tooltip>
        <TooltipTrigger
          render={
            <Button
              variant='ghost'
              size='icon-sm'
              onClick={() => props.onView(props.plan, 'participants')}
              aria-label={t('Participants')}
            />
          }
        >
          <Users />
        </TooltipTrigger>
        <TooltipContent>{t('Participants')}</TooltipContent>
      </Tooltip>

      <DataTableRowActionMenu ariaLabel={t('Open menu')} modal={false}>
        <DropdownMenuItem
          disabled={!canManage}
          onClick={() => props.onEdit(props.plan)}
        >
          {t('Edit')}
          <DropdownMenuShortcut>
            <Pencil size={16} />
          </DropdownMenuShortcut>
        </DropdownMenuItem>
        <DropdownMenuItem
          disabled={props.drawPending || props.plan.status !== 'open'}
          onClick={() => props.onDraw(props.plan)}
        >
          {t('Draw now')}
          <DropdownMenuShortcut>
            <Play size={16} />
          </DropdownMenuShortcut>
        </DropdownMenuItem>
        <DropdownMenuSeparator />
        <DropdownMenuItem
          disabled={props.cancelPending || !canManage}
          onClick={() => props.onCancel(props.plan)}
          className='text-destructive focus:text-destructive'
        >
          {t('Cancel')}
          <DropdownMenuShortcut>
            <XCircle size={16} />
          </DropdownMenuShortcut>
        </DropdownMenuItem>
      </DataTableRowActionMenu>
    </div>
  )
}
