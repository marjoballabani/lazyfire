---
layout: home

hero:
  name: LazyFire
  text: Firebase in your terminal
  tagline: Browse Firestore, monitor Cloud Functions, and view live logs - all from your terminal
  image:
    src: /logo.svg
    alt: LazyFire
  actions:
    - theme: brand
      text: Get Started
      link: /guide/getting-started
    - theme: alt
      text: View on GitHub
      link: https://github.com/marjoballabani/lazyfire

features:
  - icon:
      src: /icons/database.svg
    title: Firestore Browser
    details: Navigate collections, documents, and subcollections with an intuitive tree view. Expand nested data effortlessly.
    link: /guide/collections
  - icon:
      src: /icons/bolt.svg
    title: Cloud Functions
    details: View deployed functions, monitor status, and stream live logs. Debug in real-time without leaving the terminal.
    link: /guide/cloud-functions
  - icon:
      src: /icons/search.svg
    title: Query Builder
    details: Build Firestore queries interactively. Filter by field, set operators, order results, and execute instantly.
    link: /guide/query-builder
  - icon:
      src: /icons/shield-check.svg
    title: Collection Health Scan
    details: Scan all collections against Firestore limits. Check document size, field count, index entries, and nesting depth in one shot.
    link: /guide/collection-health-scan
  - icon:
      src: /icons/chart.svg
    title: Document Stats
    details: Monitor document size, field count, index entries, and nesting depth. Color-coded warnings for Firestore limit compliance.
    link: /guide/document-stats
  - icon:
      src: /icons/terminal.svg
    title: Vim Keybindings
    details: Navigate with familiar keys - j/k, h/l, gg, G. Designed for developers who live in the terminal.
    link: /reference/keybindings
  - icon:
      src: /icons/monitor.svg
    title: Emulator Support
    details: Connect to a local Firebase Emulator for development. Browse your local Firestore without touching production data.
    link: /guide/emulator-mode
  - icon:
      src: /icons/palette.svg
    title: Customizable Themes
    details: Configure colors via YAML. Match your terminal aesthetic with hex colors, named colors, or 256-color palette.
    link: /reference/themes
---

<div class="home-content">

## Installation

::: code-group

```bash [Homebrew (Recommended)]
brew install marjoballabani/tap/lazyfire
```

```bash [Go Install]
go install github.com/marjoballabani/lazyfire@latest
```

```bash [Download Binary]
# macOS (Apple Silicon)
curl -L https://github.com/marjoballabani/lazyfire/releases/latest/download/lazyfire_darwin_arm64.tar.gz | tar xz

# macOS (Intel)
curl -L https://github.com/marjoballabani/lazyfire/releases/latest/download/lazyfire_darwin_amd64.tar.gz | tar xz

# Linux
curl -L https://github.com/marjoballabani/lazyfire/releases/latest/download/lazyfire_linux_amd64.tar.gz | tar xz
```

:::

Then run:

```bash
lazyfire
```

## Preview

![LazyFire Preview](/preview.gif){.preview-img}

## Why LazyFire?

<script setup>
import { withBase } from 'vitepress'
</script>

<div class="why-grid">
  <div class="why-card">
    <img :src="withBase('/icons/rocket.svg')" class="why-icon" alt="">
    <div class="why-title">No context switching</div>
    <div class="why-desc">Stay in your terminal while debugging Firebase. No browser tabs, no Firebase Console clicks.</div>
  </div>
  <div class="why-card">
    <img :src="withBase('/icons/zap.svg')" class="why-icon" alt="">
    <div class="why-title">Fast navigation</div>
    <div class="why-desc">Vim-style keys mean muscle memory works here too. Navigate projects, collections, and documents in seconds.</div>
  </div>
  <div class="why-card">
    <img :src="withBase('/icons/radio.svg')" class="why-icon" alt="">
    <div class="why-title">Real-time logs</div>
    <div class="why-desc">Stream Cloud Function logs without opening the console. Filter and search through live output.</div>
  </div>
  <div class="why-card">
    <img :src="withBase('/icons/filter.svg')" class="why-icon" alt="">
    <div class="why-title">Query power</div>
    <div class="why-desc">Build and execute Firestore queries interactively. Where clauses, ordering, and limits in a simple UI.</div>
  </div>
  <div class="why-card">
    <img :src="withBase('/icons/sparkles.svg')" class="why-icon" alt="">
    <div class="why-title">Zero config</div>
    <div class="why-desc">Uses your existing firebase login credentials. Install and run -- that's it.</div>
  </div>
</div>

## Firebase Console vs LazyFire

<div class="comparison-table">

| Task | Firebase Console | LazyFire |
|------|-----------------|----------|
| Switch between projects | Click through menus | `j`/`k` + `Space` |
| Browse subcollections | Click, wait, click, wait | `l` to expand, `h` to collapse |
| Check document limits | Not available | Automatic color-coded stats |
| Scan collection health | Manual, one doc at a time | `S` scans all collections |
| Query documents | Web form with limited operators | Interactive builder with `F` |
| Copy document JSON | Select, copy, format | `c` key |
| View function logs | Separate Cloud Logging page | `]` tab, same window |

</div>

</div>
