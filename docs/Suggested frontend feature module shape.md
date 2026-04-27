Each major feature should own:
routes or pages
API hooks
feature components
feature state
feature types
tests
Example:
features/source-feeds/
├─ api/
├─ components/
├─ pages/
├─ hooks/
├─ types/
└─ tests/
This is better than a giant shared-components-first approach.