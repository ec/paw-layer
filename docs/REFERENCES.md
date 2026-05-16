# References

## clawd-on-desk

URL: https://github.com/rullerzhou-afk/clawd-on-desk

Relevant ideas:

- desktop pet as a state machine with expressive animation states
- clear mapping from external events to visual states
- theme packs and imported animation packs
- eye tracking / cursor awareness
- click reactions and drag interactions
- mini mode / edge peek behavior
- multi-monitor position memory
- transparent click-through behavior where only the pet body is interactive
- tray controls and DND mode

Not in early scope:

- agent hooks/plugins
- permission bubbles
- auto-updater
- cross-platform Electron shell
- AI-agent monitoring

Potentially useful later:

- OpenClaw/session awareness as optional plugin, not core MVP
- state mapping document for animation selection
- theme import format compatibility if simple enough

## CATAI

URL: https://github.com/wil-pe/CATAI

Relevant ideas:

- cats walk on a screen affordance, then perch on active windows when needed
- multiple cats with distinct colors/personalities
- cursor awareness: look at cursor, wake up, chase/avoid
- random speech bubbles / meows
- pixel-perfect nearest-neighbor scaling
- sprite asset organization with directions and animations
- active-window perch behavior
- careful non-intrusive bubbles that do not steal focus
- cache frontmost-window state to avoid excessive polling

Not in early scope:

- Ollama chat
- cat debate mode
- macOS Dock behavior
- 8-direction high-volume sprite set

Potentially useful later:

- multi-cat color tinting
- speech bubbles as purely visual non-focus overlays
- 8-direction sprite manifest extension

## Direction for paw-layer

Keep the core product narrower than both references:

1. Hyprland-native desktop cats first.
2. Strong compositor/window behavior: sit, peek, hide on fullscreen, react to cursor.
3. Theme/sprite format early enough that art can evolve independently.
4. Optional integrations later; no AI assistant feature in the core MVP.
