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
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { LegalConsent } from '@/features/auth/components/legal-consent'
import { OAuthProviders } from '@/features/auth/components/oauth-providers'
import { saveAffiliateCode } from '@/features/auth/lib/storage'
import { useStatus } from '@/hooks/use-status'
import { cn } from '@/lib/utils'

export function SignUpForm({
  className,
  ...props
}: React.HTMLAttributes<HTMLFormElement>) {
  const { t } = useTranslation()
  const { status } = useStatus()
  const [agreedToLegal, setAgreedToLegal] = useState(false)

  const hasUserAgreement = Boolean(
    status?.user_agreement_enabled ?? status?.data?.user_agreement_enabled
  )
  const hasPrivacyPolicy = Boolean(
    status?.privacy_policy_enabled ?? status?.data?.privacy_policy_enabled
  )
  const requiresLegalConsent = hasUserAgreement || hasPrivacyPolicy
  const registrationEnabled =
    status?.register_enabled ?? status?.data?.register_enabled ?? true
  const oauthRegistrationEnabled =
    status?.oauth_register_enabled ??
    status?.data?.oauth_register_enabled ??
    true
  const githubOAuthEnabled = Boolean(
    status?.github_oauth ?? status?.data?.github_oauth
  )
  const minimumAgeDays =
    status?.github_registration_min_age_days ??
    status?.data?.github_registration_min_age_days ??
    180

  useEffect(() => {
    setAgreedToLegal(!requiresLegalConsent)
  }, [requiresLegalConsent])

  useEffect(() => {
    const affiliateCode = new URLSearchParams(window.location.search)
      .get('aff')
      ?.trim()
    if (affiliateCode) {
      saveAffiliateCode(affiliateCode)
    }
  }, [])

  const canStartRegistration =
    registrationEnabled && oauthRegistrationEnabled && githubOAuthEnabled

  return (
    <form
      className={cn('grid gap-4', className)}
      onSubmit={(event) => event.preventDefault()}
      {...props}
    >
      <Alert>
        <AlertTitle>{t('GitHub registration required')}</AlertTitle>
        <AlertDescription>
          {minimumAgeDays > 0
            ? t(
                'New accounts must register with a GitHub account created at least {{days}} days ago.',
                { days: minimumAgeDays }
              )
            : t('New accounts must register with GitHub.')}
        </AlertDescription>
      </Alert>

      <LegalConsent
        status={status}
        checked={agreedToLegal}
        onCheckedChange={setAgreedToLegal}
      />

      {!registrationEnabled && (
        <Alert variant='destructive'>
          <AlertTitle>{t('Registration is closed')}</AlertTitle>
          <AlertDescription>
            {t('Please contact the administrator for access.')}
          </AlertDescription>
        </Alert>
      )}
      {registrationEnabled &&
        (!oauthRegistrationEnabled || !githubOAuthEnabled) && (
          <Alert variant='destructive'>
            <AlertTitle>{t('GitHub registration is unavailable')}</AlertTitle>
            <AlertDescription>
              {t('Please contact the administrator to configure GitHub OAuth.')}
            </AlertDescription>
          </Alert>
        )}

      {canStartRegistration && (
        <OAuthProviders
          registrationOnly
          status={status}
          disabled={requiresLegalConsent && !agreedToLegal}
        />
      )}
    </form>
  )
}
