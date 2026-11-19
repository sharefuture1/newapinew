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
// WendingAI (2026-11): index.html shows a static landing shell (#wd-boot) on
// "/" while the bundle downloads. The app removes it once real content is up.

export const BOOT_SHELL_ID = 'wd-boot'

export function removeBootShell(): void {
  if (typeof document === 'undefined') return
  document.getElementById(BOOT_SHELL_ID)?.remove()
}
