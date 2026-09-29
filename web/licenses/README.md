# Supplemental upstream notice

The npm publication of `react-remove-scroll-bar@2.3.8` declares MIT but omits its
license file. The original upstream notice was retrieved on 30 September 2026
from https://github.com/theKashey/react-remove-scroll-bar/blob/master/LICENSE.
Git blob SHA: `7c08c3990396ecefd90f99ff5d9a34f26f5b5616`.

The version-specific copy is retained here so builds need no network license
lookup. `scripts/notices.mjs` copies installed production dependency licenses
and uses this supplement only when the package lacks its own distributed text.
The resulting THIRD_PARTY_NOTICES.txt is embedded alongside the UI, outside the
initial JavaScript bundle. A dependency upgrade without required license text
fails the build for review.
