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
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Plus } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { SectionPageLayout } from '@/components/layout'
import { Button } from '@/components/ui/button'

import { cancelLotteryPlan, drawLotteryPlan, getAdminLotteryPlans } from './api'
import { LotteryPlanCreateDrawer } from './components/lottery-plan-create-drawer'
import {
  LotteryPlanDetailsDrawer,
  type LotteryDetailsTab,
} from './components/lottery-plan-details-drawer'
import { LotteryPlanEditDrawer } from './components/lottery-plan-edit-drawer'
import { LotteryPlansTable } from './components/lottery-plans-table'
import type { LotteryPlan } from './types'

export function LotteryAdmin() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [createOpen, setCreateOpen] = useState(false)
  const [detailsOpen, setDetailsOpen] = useState(false)
  const [detailsTab, setDetailsTab] = useState<LotteryDetailsTab>('overview')
  const [selectedPlan, setSelectedPlan] = useState<LotteryPlan | null>(null)
  const [editingPlan, setEditingPlan] = useState<LotteryPlan | null>(null)

  const plansQuery = useQuery({
    queryKey: ['lottery', 'admin', 'plans'],
    queryFn: getAdminLotteryPlans,
  })
  const plans = plansQuery.data?.success ? (plansQuery.data.data ?? []) : []

  const drawMutation = useMutation({
    mutationFn: ({ planId, reason }: { planId: number; reason: string }) =>
      drawLotteryPlan(planId, reason),
    onSuccess: async (result) => {
      if (!result.success) {
        toast.error(t('Failed to draw lottery'))
        return
      }
      toast.success(t('Lottery draw completed'))
      await queryClient.invalidateQueries({
        queryKey: ['lottery', 'admin', 'plans'],
      })
    },
  })
  const cancelMutation = useMutation({
    mutationFn: cancelLotteryPlan,
    onSuccess: async (result) => {
      if (!result.success) {
        toast.error(t('Failed to cancel lottery plan'))
        return
      }
      toast.success(t('Lottery plan cancelled'))
      await queryClient.invalidateQueries({
        queryKey: ['lottery', 'admin', 'plans'],
      })
    },
  })

  const viewPlan = (plan: LotteryPlan, tab: LotteryDetailsTab): void => {
    setSelectedPlan(plan)
    setDetailsTab(tab)
    setDetailsOpen(true)
  }

  const requestManualDraw = (plan: LotteryPlan): void => {
    const reason = window.prompt(t('Enter the reason for the early draw'))
    if (!reason?.trim()) return
    if (!window.confirm(t('Confirm early draw? This cannot be undone.'))) return
    drawMutation.mutate({ planId: plan.id, reason: reason.trim() })
  }

  const requestCancel = (plan: LotteryPlan): void => {
    if (!window.confirm(t('Cancel this lottery plan?'))) return
    cancelMutation.mutate(plan.id)
  }

  return (
    <>
      <SectionPageLayout fixedContent>
        <SectionPageLayout.Title>
          {t('Lottery Management')}
        </SectionPageLayout.Title>
        <SectionPageLayout.Actions>
          <Button size='sm' onClick={() => setCreateOpen(true)}>
            <Plus />
            {t('Create lottery plan')}
          </Button>
        </SectionPageLayout.Actions>
        <SectionPageLayout.Content>
          <LotteryPlansTable
            plans={plans}
            isLoading={plansQuery.isLoading}
            isFetching={plansQuery.isFetching}
            cancelPending={cancelMutation.isPending}
            drawPending={drawMutation.isPending}
            onCreate={() => setCreateOpen(true)}
            onView={viewPlan}
            onEdit={setEditingPlan}
            onDraw={requestManualDraw}
            onCancel={requestCancel}
          />
        </SectionPageLayout.Content>
      </SectionPageLayout>

      <LotteryPlanCreateDrawer open={createOpen} onOpenChange={setCreateOpen} />
      <LotteryPlanDetailsDrawer
        open={detailsOpen}
        plan={selectedPlan}
        initialTab={detailsTab}
        onOpenChange={setDetailsOpen}
      />
      <LotteryPlanEditDrawer
        open={editingPlan !== null}
        plan={editingPlan}
        onOpenChange={(open) => {
          if (!open) setEditingPlan(null)
        }}
      />
    </>
  )
}
