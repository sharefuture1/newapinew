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
import { useTranslation } from 'react-i18next'
import { z } from 'zod'

import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormLabel,
} from '@/components/ui/form'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import { THEME_PRESETS } from '@/lib/theme-customization'

import { SettingsPageFormActions } from '../components/settings-page-context'
import { SettingsSection } from '../components/settings-section'
import {
  SettingsForm,
  SettingsSwitchContent,
  SettingsSwitchItem,
} from '../components/settings-form-layout'
import { useSettingsForm } from '../hooks/use-settings-form'
import { useUpdateOption } from '../hooks/use-update-option'

const appearanceSchema = z.object({
  appearance_setting: z.object({
    theme_mode: z.enum(['light', 'dark', 'system']),
    theme_preset: z.string().max(64),
    allow_user_theme: z.boolean(),
  }),
})

export type AppearanceFormValues = z.infer<typeof appearanceSchema>

interface AppearanceSectionProps {
  defaultValues: AppearanceFormValues
}

export function AppearanceSection(props: AppearanceSectionProps) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()

  const { form, handleSubmit } = useSettingsForm({
    resolver: zodResolver(appearanceSchema),
    defaultValues: props.defaultValues,
    onSubmit: async (_data, changed) => {
      for (const [key, value] of Object.entries(changed)) {
        await updateOption.mutateAsync({
          key,
          value: value as string | number | boolean,
        })
      }
    },
  })

  return (
    <SettingsSection title={t('Appearance')}>
      <Form {...form}>
        <SettingsForm onSubmit={handleSubmit}>
          <SettingsPageFormActions
            onSave={handleSubmit}
            isSaving={updateOption.isPending}
          />

          <FormField
            control={form.control}
            name='appearance_setting.theme_mode'
            render={({ field }) => (
              <div className='space-y-2'>
                <FormLabel>{t('Theme mode')}</FormLabel>
                <FormControl>
                  <Select
                    items={[
                      { value: 'light', label: t('Light') },
                      { value: 'dark', label: t('Dark') },
                      { value: 'system', label: t('System') },
                    ]}
                    value={field.value}
                    onValueChange={field.onChange}
                  >
                    <SelectTrigger aria-label={t('Theme mode')}>
                      <SelectValue placeholder={t('Select theme mode')} />
                    </SelectTrigger>
                    <SelectContent alignItemWithTrigger={false}>
                      <SelectGroup>
                        <SelectItem value='light'>{t('Light')}</SelectItem>
                        <SelectItem value='dark'>{t('Dark')}</SelectItem>
                        <SelectItem value='system'>{t('System')}</SelectItem>
                      </SelectGroup>
                    </SelectContent>
                  </Select>
                </FormControl>
                <FormDescription>
                  {t('Applied to every user and to visitors before sign-in.')}
                </FormDescription>
              </div>
            )}
          />

          <FormField
            control={form.control}
            name='appearance_setting.theme_preset'
            render={({ field }) => (
              <div className='space-y-2'>
                <FormLabel>{t('Color preset')}</FormLabel>
                <FormControl>
                  <Select
                    items={THEME_PRESETS.map((preset) => ({
                      value: preset.value,
                      label: t(`preset.${preset.value}`),
                    }))}
                    value={field.value}
                    onValueChange={field.onChange}
                  >
                    <SelectTrigger aria-label={t('Color preset')}>
                      <SelectValue placeholder={t('Select color preset')} />
                    </SelectTrigger>
                    <SelectContent alignItemWithTrigger={false}>
                      <SelectGroup>
                        {THEME_PRESETS.map((preset) => (
                          <SelectItem key={preset.value} value={preset.value}>
                            {t(`preset.${preset.value}`)}
                          </SelectItem>
                        ))}
                      </SelectGroup>
                    </SelectContent>
                  </Select>
                </FormControl>
                <FormDescription>
                  {t('Preset styles are defined in the frontend theme CSS.')}
                </FormDescription>
              </div>
            )}
          />

          <FormField
            control={form.control}
            name='appearance_setting.allow_user_theme'
            render={({ field }) => (
              <SettingsSwitchItem>
                <SettingsSwitchContent>
                  <FormLabel>{t('Allow users to switch theme')}</FormLabel>
                  <FormDescription>
                    {t(
                      'When disabled, the theme and color preset above are forced for everyone.'
                    )}
                  </FormDescription>
                </SettingsSwitchContent>
                <FormControl>
                  <Switch
                    aria-label={t('Allow users to switch theme')}
                    checked={field.value}
                    onCheckedChange={field.onChange}
                  />
                </FormControl>
              </SettingsSwitchItem>
            )}
          />
        </SettingsForm>
      </Form>
    </SettingsSection>
  )
}
