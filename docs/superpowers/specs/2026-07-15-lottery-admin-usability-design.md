# Lottery Admin Usability Design

## Goal

Make lottery creation understandable in Chinese, replace manual group/user ID entry with searchable selectors, and ensure plans can be created on MySQL without the `draw_algorithm` truncation error.

## Root Causes

- Lottery and extended redemption-code screens use translation keys that are absent from locale files, so i18next falls back to English source text.
- `LotteryAdmin` stores group names and user IDs in one free-form `allow_list` string.
- The creation form is a dense unlabeled grid. Numeric fields such as quota and claim expiry have no units or explanations.
- `weighted_random_without_replacement` is longer than the MySQL `VARCHAR(32)` column used by `LotteryPlan.DrawAlgorithm`.

## Interface Design

The admin page remains a dense operational page rather than a wizard. The creation form is split into three full-width sections:

1. **Basic information**: title, description, maximum participants, registration start time, and draw time.
2. **Participation eligibility**: all users, selected groups, or selected users. Group and user selections use searchable multi-select controls with selected chips.
3. **Prize settings**: each prize has explicit labels. Quota fields are shown only for quota rewards, and subscription-plan fields only for subscription rewards. Claim expiry is entered in days and converted to seconds in the API payload.

Every field has a visible label. Fields whose meaning is not obvious also include concise helper text and validation errors. The plan list translates statuses and displays schedule, eligibility mode, capacity, and available actions in a scannable layout.

## Data Flow

- Groups come from the existing admin endpoint `GET /api/group/`.
- User search uses `GET /api/user/search`, returning up to 50 matches for username, display name, email, or ID search.
- The form stores `selected_groups: string[]` and `selected_user_ids: number[]`; it no longer parses free-form IDs.
- Submission preserves the existing lottery API contract: `groups` and `user_ids` arrays.

## Database Compatibility

Use a stable algorithm identifier that fits the legacy 32-character column and widen the GORM column definition to `VARCHAR(64)` for future identifiers. Existing databases migrate on application startup, while the shorter identifier prevents insertion failure even before the alteration is applied.

## Internationalization

All lottery and redemption-code strings used by the changed screens are added to every supported locale through the repository translation script workflow. Chinese and Traditional Chinese receive native translations; other locales receive reviewed translations rather than English fallback values.

## Verification

- Backend regression test verifies the default draw algorithm remains compatible with the legacy column and is persisted during plan creation.
- Frontend unit tests cover form-to-payload conversion for groups, users, reward types, and claim-expiry days.
- Run targeted Go tests, frontend tests, TypeScript typecheck, lint, formatting checks, i18n sync, and production build.
