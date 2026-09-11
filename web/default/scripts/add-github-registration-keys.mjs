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
import fs from 'node:fs/promises'
import path from 'node:path'

const translations = {
  en: {
    'GitHub registration required': 'GitHub registration required',
    'New accounts must register with a GitHub account created at least {{days}} days ago.':
      'New accounts must register with a GitHub account created at least {{days}} days ago.',
    'New accounts must register with GitHub.':
      'New accounts must register with GitHub.',
    'Please contact the administrator for access.':
      'Please contact the administrator for access.',
    'GitHub registration is unavailable': 'GitHub registration is unavailable',
    'Please contact the administrator to configure GitHub OAuth.':
      'Please contact the administrator to configure GitHub OAuth.',
    'Minimum GitHub account age (days)': 'Minimum GitHub account age (days)',
    'New GitHub accounts younger than this are rejected. Set 0 to disable the age limit.':
      'New GitHub accounts younger than this are rejected. Set 0 to disable the age limit.',
    'GitHub account age exemptions': 'GitHub account age exemptions',
    'Exemptions are matched by the stable numeric GitHub ID and only bypass the account-age requirement.':
      'Exemptions are matched by the stable numeric GitHub ID and only bypass the account-age requirement.',
    'GitHub username': 'GitHub username',
    'Resolve identity': 'Resolve identity',
    'Unable to verify GitHub account': 'Unable to verify GitHub account',
    'Failed to add exemption': 'Failed to add exemption',
    'GitHub age exemption added': 'GitHub age exemption added',
    'Failed to update remark': 'Failed to update remark',
    'Remark updated': 'Remark updated',
    'Failed to remove exemption': 'Failed to remove exemption',
    'GitHub age exemption removed': 'GitHub age exemption removed',
    'Confirm GitHub identity': 'Confirm GitHub identity',
    'Numeric ID': 'Numeric ID',
    'Account created': 'Account created',
    'Reason for exemption': 'Reason for exemption',
    'Confirm and add exemption': 'Confirm and add exemption',
    'GitHub account': 'GitHub account',
    'No GitHub age exemptions': 'No GitHub age exemptions',
    'Remark for {{username}}': 'Remark for {{username}}',
    'Remove exemption for {{username}}': 'Remove exemption for {{username}}',
    'Remove GitHub age exemption?': 'Remove GitHub age exemption?',
    'This only removes the age exemption. It does not change existing users.':
      'This only removes the age exemption. It does not change existing users.',
  },
  zh: {
    'GitHub registration required': '需要通过 GitHub 注册',
    'New accounts must register with a GitHub account created at least {{days}} days ago.':
      '新账号必须使用注册时间至少满 {{days}} 天的 GitHub 账号。',
    'New accounts must register with GitHub.': '新账号必须通过 GitHub 注册。',
    'Please contact the administrator for access.':
      '请联系管理员获取访问权限。',
    'GitHub registration is unavailable': 'GitHub 注册暂不可用',
    'Please contact the administrator to configure GitHub OAuth.':
      '请联系管理员配置 GitHub OAuth。',
    'Minimum GitHub account age (days)': 'GitHub 账号最短注册天数',
    'New GitHub accounts younger than this are rejected. Set 0 to disable the age limit.':
      '注册天数不足的 GitHub 账号将被拒绝；设为 0 可关闭年龄限制。',
    'GitHub account age exemptions': 'GitHub 账号年龄豁免',
    'Exemptions are matched by the stable numeric GitHub ID and only bypass the account-age requirement.':
      '豁免按稳定的 GitHub 数字 ID 匹配，并且仅绕过账号年龄要求。',
    'GitHub username': 'GitHub 用户名',
    'Resolve identity': '解析身份',
    'Unable to verify GitHub account': '无法验证 GitHub 账号',
    'Failed to add exemption': '添加豁免失败',
    'GitHub age exemption added': '已添加 GitHub 年龄豁免',
    'Failed to update remark': '更新备注失败',
    'Remark updated': '备注已更新',
    'Failed to remove exemption': '移除豁免失败',
    'GitHub age exemption removed': '已移除 GitHub 年龄豁免',
    'Confirm GitHub identity': '确认 GitHub 身份',
    'Numeric ID': '数字 ID',
    'Account created': '账号创建时间',
    'Reason for exemption': '豁免原因',
    'Confirm and add exemption': '确认并添加豁免',
    'GitHub account': 'GitHub 账号',
    'No GitHub age exemptions': '暂无 GitHub 年龄豁免',
    'Remark for {{username}}': '{{username}} 的备注',
    'Remove exemption for {{username}}': '移除 {{username}} 的豁免',
    'Remove GitHub age exemption?': '移除 GitHub 年龄豁免？',
    'This only removes the age exemption. It does not change existing users.':
      '此操作只会移除年龄豁免，不会更改现有用户。',
  },
  'zh-TW': {
    'GitHub registration required': '需要透過 GitHub 註冊',
    'New accounts must register with a GitHub account created at least {{days}} days ago.':
      '新帳號必須使用註冊時間至少滿 {{days}} 天的 GitHub 帳號。',
    'New accounts must register with GitHub.': '新帳號必須透過 GitHub 註冊。',
    'Please contact the administrator for access.':
      '請聯絡管理員取得存取權限。',
    'GitHub registration is unavailable': 'GitHub 註冊目前無法使用',
    'Please contact the administrator to configure GitHub OAuth.':
      '請聯絡管理員設定 GitHub OAuth。',
    'Minimum GitHub account age (days)': 'GitHub 帳號最短註冊天數',
    'New GitHub accounts younger than this are rejected. Set 0 to disable the age limit.':
      '註冊天數不足的 GitHub 帳號將被拒絕；設為 0 可停用年齡限制。',
    'GitHub account age exemptions': 'GitHub 帳號年齡豁免',
    'Exemptions are matched by the stable numeric GitHub ID and only bypass the account-age requirement.':
      '豁免依穩定的 GitHub 數字 ID 比對，且只略過帳號年齡要求。',
    'GitHub username': 'GitHub 使用者名稱',
    'Resolve identity': '解析身分',
    'Unable to verify GitHub account': '無法驗證 GitHub 帳號',
    'Failed to add exemption': '新增豁免失敗',
    'GitHub age exemption added': '已新增 GitHub 年齡豁免',
    'Failed to update remark': '更新備註失敗',
    'Remark updated': '備註已更新',
    'Failed to remove exemption': '移除豁免失敗',
    'GitHub age exemption removed': '已移除 GitHub 年齡豁免',
    'Confirm GitHub identity': '確認 GitHub 身分',
    'Numeric ID': '數字 ID',
    'Account created': '帳號建立時間',
    'Reason for exemption': '豁免原因',
    'Confirm and add exemption': '確認並新增豁免',
    'GitHub account': 'GitHub 帳號',
    'No GitHub age exemptions': '目前沒有 GitHub 年齡豁免',
    'Remark for {{username}}': '{{username}} 的備註',
    'Remove exemption for {{username}}': '移除 {{username}} 的豁免',
    'Remove GitHub age exemption?': '移除 GitHub 年齡豁免？',
    'This only removes the age exemption. It does not change existing users.':
      '此操作只會移除年齡豁免，不會變更現有使用者。',
  },
  fr: {
    'GitHub registration required': 'Inscription GitHub requise',
    'New accounts must register with a GitHub account created at least {{days}} days ago.':
      'Les nouveaux comptes doivent utiliser un compte GitHub créé il y a au moins {{days}} jours.',
    'New accounts must register with GitHub.':
      'Les nouveaux comptes doivent s’inscrire avec GitHub.',
    'Please contact the administrator for access.':
      'Contactez l’administrateur pour obtenir l’accès.',
    'GitHub registration is unavailable': 'Inscription GitHub indisponible',
    'Please contact the administrator to configure GitHub OAuth.':
      'Contactez l’administrateur pour configurer GitHub OAuth.',
    'Minimum GitHub account age (days)': 'Âge minimal du compte GitHub (jours)',
    'New GitHub accounts younger than this are rejected. Set 0 to disable the age limit.':
      'Les comptes GitHub plus récents sont refusés. Définissez 0 pour désactiver cette limite.',
    'GitHub account age exemptions': 'Dérogations d’âge GitHub',
    'Exemptions are matched by the stable numeric GitHub ID and only bypass the account-age requirement.':
      'Les dérogations utilisent l’identifiant GitHub numérique stable et contournent uniquement la condition d’âge.',
    'GitHub username': 'Nom d’utilisateur GitHub',
    'Resolve identity': 'Vérifier l’identité',
    'Unable to verify GitHub account':
      'Impossible de vérifier le compte GitHub',
    'Failed to add exemption': 'Échec de l’ajout de la dérogation',
    'GitHub age exemption added': 'Dérogation d’âge GitHub ajoutée',
    'Failed to update remark': 'Échec de la mise à jour de la remarque',
    'Remark updated': 'Remarque mise à jour',
    'Failed to remove exemption': 'Échec de la suppression de la dérogation',
    'GitHub age exemption removed': 'Dérogation d’âge GitHub supprimée',
    'Confirm GitHub identity': 'Confirmer l’identité GitHub',
    'Numeric ID': 'ID numérique',
    'Account created': 'Compte créé',
    'Reason for exemption': 'Motif de la dérogation',
    'Confirm and add exemption': 'Confirmer et ajouter',
    'GitHub account': 'Compte GitHub',
    'No GitHub age exemptions': 'Aucune dérogation d’âge GitHub',
    'Remark for {{username}}': 'Remarque pour {{username}}',
    'Remove exemption for {{username}}':
      'Supprimer la dérogation de {{username}}',
    'Remove GitHub age exemption?': 'Supprimer la dérogation d’âge GitHub ?',
    'This only removes the age exemption. It does not change existing users.':
      'Cela supprime uniquement la dérogation d’âge et ne modifie pas les utilisateurs existants.',
  },
  ja: {
    'GitHub registration required': 'GitHub 登録が必要です',
    'New accounts must register with a GitHub account created at least {{days}} days ago.':
      '新規アカウントは、作成から {{days}} 日以上経過した GitHub アカウントで登録する必要があります。',
    'New accounts must register with GitHub.':
      '新規アカウントは GitHub で登録する必要があります。',
    'Please contact the administrator for access.':
      'アクセスについては管理者にお問い合わせください。',
    'GitHub registration is unavailable': 'GitHub 登録を利用できません',
    'Please contact the administrator to configure GitHub OAuth.':
      'GitHub OAuth の設定について管理者にお問い合わせください。',
    'Minimum GitHub account age (days)': 'GitHub アカウントの最低経過日数',
    'New GitHub accounts younger than this are rejected. Set 0 to disable the age limit.':
      'これより新しい GitHub アカウントは拒否されます。0 にすると制限を無効化します。',
    'GitHub account age exemptions': 'GitHub アカウント年齢の免除',
    'Exemptions are matched by the stable numeric GitHub ID and only bypass the account-age requirement.':
      '免除は安定した GitHub 数値 ID で照合され、アカウント年齢要件のみを回避します。',
    'GitHub username': 'GitHub ユーザー名',
    'Resolve identity': '本人情報を確認',
    'Unable to verify GitHub account': 'GitHub アカウントを確認できません',
    'Failed to add exemption': '免除の追加に失敗しました',
    'GitHub age exemption added': 'GitHub 年齢免除を追加しました',
    'Failed to update remark': '備考の更新に失敗しました',
    'Remark updated': '備考を更新しました',
    'Failed to remove exemption': '免除の削除に失敗しました',
    'GitHub age exemption removed': 'GitHub 年齢免除を削除しました',
    'Confirm GitHub identity': 'GitHub 本人情報を確認',
    'Numeric ID': '数値 ID',
    'Account created': 'アカウント作成日',
    'Reason for exemption': '免除理由',
    'Confirm and add exemption': '確認して免除を追加',
    'GitHub account': 'GitHub アカウント',
    'No GitHub age exemptions': 'GitHub 年齢免除はありません',
    'Remark for {{username}}': '{{username}} の備考',
    'Remove exemption for {{username}}': '{{username}} の免除を削除',
    'Remove GitHub age exemption?': 'GitHub 年齢免除を削除しますか？',
    'This only removes the age exemption. It does not change existing users.':
      '年齢免除のみを削除し、既存ユーザーは変更しません。',
  },
  ru: {
    'GitHub registration required': 'Требуется регистрация через GitHub',
    'New accounts must register with a GitHub account created at least {{days}} days ago.':
      'Новые пользователи должны регистрироваться через аккаунт GitHub возрастом не менее {{days}} дней.',
    'New accounts must register with GitHub.':
      'Новые пользователи должны регистрироваться через GitHub.',
    'Please contact the administrator for access.':
      'Для получения доступа обратитесь к администратору.',
    'GitHub registration is unavailable': 'Регистрация через GitHub недоступна',
    'Please contact the administrator to configure GitHub OAuth.':
      'Обратитесь к администратору для настройки GitHub OAuth.',
    'Minimum GitHub account age (days)':
      'Минимальный возраст аккаунта GitHub (дни)',
    'New GitHub accounts younger than this are rejected. Set 0 to disable the age limit.':
      'Более новые аккаунты GitHub отклоняются. Укажите 0, чтобы отключить ограничение.',
    'GitHub account age exemptions': 'Исключения по возрасту GitHub',
    'Exemptions are matched by the stable numeric GitHub ID and only bypass the account-age requirement.':
      'Исключения сопоставляются по стабильному числовому ID GitHub и отменяют только требование к возрасту.',
    'GitHub username': 'Имя пользователя GitHub',
    'Resolve identity': 'Проверить личность',
    'Unable to verify GitHub account': 'Не удалось проверить аккаунт GitHub',
    'Failed to add exemption': 'Не удалось добавить исключение',
    'GitHub age exemption added': 'Исключение по возрасту GitHub добавлено',
    'Failed to update remark': 'Не удалось обновить примечание',
    'Remark updated': 'Примечание обновлено',
    'Failed to remove exemption': 'Не удалось удалить исключение',
    'GitHub age exemption removed': 'Исключение по возрасту GitHub удалено',
    'Confirm GitHub identity': 'Подтвердите личность GitHub',
    'Numeric ID': 'Числовой ID',
    'Account created': 'Аккаунт создан',
    'Reason for exemption': 'Причина исключения',
    'Confirm and add exemption': 'Подтвердить и добавить',
    'GitHub account': 'Аккаунт GitHub',
    'No GitHub age exemptions': 'Нет исключений по возрасту GitHub',
    'Remark for {{username}}': 'Примечание для {{username}}',
    'Remove exemption for {{username}}': 'Удалить исключение для {{username}}',
    'Remove GitHub age exemption?': 'Удалить исключение по возрасту GitHub?',
    'This only removes the age exemption. It does not change existing users.':
      'Будет удалено только исключение по возрасту; существующие пользователи не изменятся.',
  },
  vi: {
    'GitHub registration required': 'Bắt buộc đăng ký qua GitHub',
    'New accounts must register with a GitHub account created at least {{days}} days ago.':
      'Tài khoản mới phải đăng ký bằng tài khoản GitHub đã được tạo ít nhất {{days}} ngày.',
    'New accounts must register with GitHub.':
      'Tài khoản mới phải đăng ký qua GitHub.',
    'Please contact the administrator for access.':
      'Vui lòng liên hệ quản trị viên để được cấp quyền truy cập.',
    'GitHub registration is unavailable': 'Đăng ký GitHub hiện không khả dụng',
    'Please contact the administrator to configure GitHub OAuth.':
      'Vui lòng liên hệ quản trị viên để cấu hình GitHub OAuth.',
    'Minimum GitHub account age (days)':
      'Tuổi tối thiểu của tài khoản GitHub (ngày)',
    'New GitHub accounts younger than this are rejected. Set 0 to disable the age limit.':
      'Tài khoản GitHub mới hơn giới hạn này sẽ bị từ chối. Đặt 0 để tắt giới hạn.',
    'GitHub account age exemptions': 'Miễn giới hạn tuổi tài khoản GitHub',
    'Exemptions are matched by the stable numeric GitHub ID and only bypass the account-age requirement.':
      'Miễn trừ được khớp bằng ID GitHub dạng số ổn định và chỉ bỏ qua yêu cầu về tuổi tài khoản.',
    'GitHub username': 'Tên người dùng GitHub',
    'Resolve identity': 'Xác minh danh tính',
    'Unable to verify GitHub account': 'Không thể xác minh tài khoản GitHub',
    'Failed to add exemption': 'Không thể thêm miễn trừ',
    'GitHub age exemption added': 'Đã thêm miễn trừ tuổi GitHub',
    'Failed to update remark': 'Không thể cập nhật ghi chú',
    'Remark updated': 'Đã cập nhật ghi chú',
    'Failed to remove exemption': 'Không thể xóa miễn trừ',
    'GitHub age exemption removed': 'Đã xóa miễn trừ tuổi GitHub',
    'Confirm GitHub identity': 'Xác nhận danh tính GitHub',
    'Numeric ID': 'ID dạng số',
    'Account created': 'Ngày tạo tài khoản',
    'Reason for exemption': 'Lý do miễn trừ',
    'Confirm and add exemption': 'Xác nhận và thêm miễn trừ',
    'GitHub account': 'Tài khoản GitHub',
    'No GitHub age exemptions': 'Không có miễn trừ tuổi GitHub',
    'Remark for {{username}}': 'Ghi chú cho {{username}}',
    'Remove exemption for {{username}}': 'Xóa miễn trừ cho {{username}}',
    'Remove GitHub age exemption?': 'Xóa miễn trừ tuổi GitHub?',
    'This only removes the age exemption. It does not change existing users.':
      'Thao tác này chỉ xóa miễn trừ tuổi và không thay đổi người dùng hiện có.',
  },
}

const localeDirectory = path.resolve('src/i18n/locales')

for (const [locale, entries] of Object.entries(translations)) {
  const filename = path.join(localeDirectory, `${locale}.json`)
  const document = JSON.parse(await fs.readFile(filename, 'utf8'))
  document.translation ??= {}
  for (const [key, value] of Object.entries(entries)) {
    document.translation[key] = value
  }
  await fs.writeFile(filename, `${JSON.stringify(document, null, 2)}\n`, 'utf8')
}

console.log('Added GitHub registration translations.')
