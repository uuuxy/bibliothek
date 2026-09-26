## 2023-10-26 - Dynamic titles for disabled buttons
**Learning:** Disabled buttons often lack context for screen reader users and mouse hover users, leaving them guessing why an action is unavailable. Using a static title (like "Nach oben") when the button is disabled can be confusing.
**Action:** When creating or modifying disabled buttons, ensure they have a dynamic `title` attribute that explains the reason for being disabled (e.g., `title={isFirst ? 'Already at top' : 'Move up'}`). This provides clear feedback and improves accessibility.
