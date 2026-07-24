# GNOME App (`whatsd-gnome`) — AGENTS.md Companion

## Overview

`whatsd-gnome` is the native GTK4 / LibAdwaita Linux desktop application for `whatsd` (the Linux-first WhatsApp daemon). It provides a responsive desktop UI built with GJS, TypeScript, gjsify, and Blueprint (`.blp`) UI templates.

The application communicates with the backend `whatsd` Go daemon via local Unix domain sockets (`/tmp/whatsd.sock`), providing chat navigation, real-time message updates, status indicators, and device pairing setup.

---

## Codebase Architecture & Structure

```
gnome/
├── package.json          # Node dependencies, build & check scripts
├── tsconfig.json         # Strict TypeScript configuration with @girs types
├── README.md             # Secondary AGENTS.md for the TS/GNOME codebase
└── src/
    ├── index.ts          # Adw.Application entrypoint
    ├── window.ts         # MainWindow GObject class & split-view logic
    ├── window.blp        # Main window layout in Blueprint DSL
    ├── types.ts          # Core domain models (Chat, Message, Contact, State)
    ├── mock-data.ts       # Dummy dataset for UI refinement & state simulation
    ├── style.css         # Custom Gtk 4 Adwaita CSS rules
    └── components/
        ├── chat-row.ts       # Sidebar Chat row item (Gtk.ListBoxRow + Adw.Avatar)
        ├── message-bubble.ts # Message bubble item (incoming / outgoing layout)
        ├── about-dialog.ts   # Adw.AboutDialog presenter
        └── qr-dialog.ts      # Adw.Dialog pair device modal presenter
```

---

## Development Setup & Workflow Commands

Ensure `@gjsify/cli` and `blueprint-compiler` are installed on your Linux system.

```bash
# 1. Install dependencies
npm install

# 2. Type-check TypeScript codebase (Strict zero-any policy)
npm run check             # Runs: gjsify tsc --noEmit

# 3. Lint & compile Blueprint (.blp) UI files
npm run lint:blp          # Runs: blueprint-compiler compile --output /dev/null src/window.blp

# 4. Build application bundle
npm run build            # Runs: gjsify build src/index.ts --outfile dist/index.js

# 5. Run built desktop application
npm start                # Runs: gjsify run dist/index.js

# 6. Combined build + dev run
npm run dev              # Builds bundle and runs the app
```

---

## Development Principles & Best Practices

1. **Strict Type Safety & Zero `any`**:
   - Avoid using `any` or explicit type casts (`as any`, `as unknown`).
   - Use explicit `@girs/*` types for GTK, LibAdwaita, Gio, and GObject APIs.
   - Run `npm run check` after every modification and ensure zero type errors.

2. **Blueprint UI Declarative DSL**:
   - UI templates are defined in Blueprint (`.blp`) files and compiled via `blueprint-compiler`.
   - Validate `.blp` files using `npm run lint:blp` before committing changes.
   - Use `InternalChildren` array inside GObject static initializer to automatically bind blueprint element IDs to class fields (prefixed with `_`).

3. **LibAdwaita Design Guidelines**:
   - Use standard LibAdwaita components (`Adw.NavigationSplitView`, `Adw.HeaderBar`, `Adw.Banner`, `Adw.StatusPage`, `Adw.Avatar`, `Adw.AboutDialog`).
   - Maintain HIG compliance (Adaptive split view, standard action buttons, dark mode support out of the box).
   - Use CSS classes from Adwaita (`accent`, `suggested-action`, `card`, `navigation-sidebar`) rather than inline static pixel hacks.

4. **Modular File Design**:
   - Keep files small, focused, and under 250 lines when practical.
   - Separate data models (`types.ts`), mock data (`mock-data.ts`), components (`components/`), and main windows (`window.ts`).

5. **AI-Assisted TS Workflow**:
   - After implementing UI behavior or component changes, always run `npm run check`, `npm run lint:blp`, and `npm run build` in a loop until clean.
   - Handle GObject signals and callbacks gracefully without swallowing exceptions.

---

## Upstream Release Train & Compatibility

All `@gjsify/*` packages ship as one release train. To upgrade gjsify dependencies across the workspace:

```bash
gjsify upgrade --latest --filter @gjsify
```
