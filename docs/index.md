---
layout: home

hero:
  name: LazyFire
  text: Firebase in your terminal
  tagline: Browse Firestore, monitor Cloud Functions, and view live logs — all from your terminal
  image:
    src: /logo.svg
    alt: LazyFire
  actions:
    - theme: brand
      text: Get Started →
      link: /guide/getting-started
    - theme: alt
      text: View on GitHub
      link: https://github.com/marjoballabani/lazyfire

features:
  - icon:
      src: /icons/database.svg
    title: Firestore Browser
    details: Navigate collections, documents, and subcollections with an intuitive tree view. Expand nested data effortlessly.
  - icon:
      src: /icons/bolt.svg
    title: Cloud Functions
    details: View deployed functions, monitor status, and stream live logs. Debug in real-time without leaving the terminal.
  - icon:
      src: /icons/search.svg
    title: Query Builder
    details: Build Firestore queries interactively. Filter by field, set operators, order results, and execute instantly.
  - icon:
      src: /icons/terminal.svg
    title: Vim Keybindings
    details: Navigate with familiar keys — j/k, h/l, gg, G. Designed for developers who live in the terminal.
  - icon:
      src: /icons/chart.svg
    title: Document Stats
    details: Monitor document size, field count, and nesting depth. Color-coded warnings for Firestore limit compliance.
  - icon:
      src: /icons/palette.svg
    title: Customizable Themes
    details: Configure colors via YAML. Match your terminal aesthetic with hex colors, named colors, or 256-color palette.
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
    <div class="why-desc">Stay in your terminal while debugging Firebase</div>
  </div>
  <div class="why-card">
    <img :src="withBase('/icons/zap.svg')" class="why-icon" alt="">
    <div class="why-title">Fast navigation</div>
    <div class="why-desc">Vim-style keys mean muscle memory works here too</div>
  </div>
  <div class="why-card">
    <img :src="withBase('/icons/radio.svg')" class="why-icon" alt="">
    <div class="why-title">Real-time logs</div>
    <div class="why-desc">Stream Cloud Function logs without opening the console</div>
  </div>
  <div class="why-card">
    <img :src="withBase('/icons/filter.svg')" class="why-icon" alt="">
    <div class="why-title">Query power</div>
    <div class="why-desc">Build and execute Firestore queries interactively</div>
  </div>
  <div class="why-card">
    <img :src="withBase('/icons/sparkles.svg')" class="why-icon" alt="">
    <div class="why-title">Zero config</div>
    <div class="why-desc">Uses your existing firebase login credentials</div>
  </div>
</div>

</div>
