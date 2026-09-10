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

export const lotteryQueryKeys = {
  plans: (userId: number) => ['lottery', 'self', userId] as const,
  results: (userId: number) => ['lottery', 'results', 'page', userId] as const,
  pendingResults: (userId: number) =>
    ['lottery', 'results', 'pending', userId] as const,
  notificationsSummary: (userId: number) =>
    ['lottery', 'notifications', 'summary', 'unread', userId] as const,
  notificationsPage: (userId: number) =>
    ['lottery', 'notifications', 'page', 'unread', userId] as const,
  userPlan: (userId: number, planId: number) =>
    ['lottery', 'self', 'plan', userId, planId] as const,
  userParticipants: (userId: number, planId: number) =>
    ['lottery', 'self', 'participants', 'page', userId, planId] as const,
  userResults: (userId: number, planId: number) =>
    ['lottery', 'self', 'plan-results', 'page', userId, planId] as const,
  adminPlan: (planId: number) => ['lottery', 'admin', 'plan', planId] as const,
} as const
