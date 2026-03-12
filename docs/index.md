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

<script setup>
import { withBase } from 'vitepress'
</script>

<div class="home-content">

<div class="section-divider"></div>

## Quick Start

<div class="quick-start">
  <div class="terminal-window">
    <div class="terminal-header">
      <div class="terminal-dot red"></div>
      <div class="terminal-dot yellow"></div>
      <div class="terminal-dot green"></div>
      <span class="terminal-title">Terminal</span>
    </div>
    <div class="terminal-body">
      <div class="terminal-line">
        <span class="terminal-prompt">$</span>
        <span class="terminal-cmd">brew install marjoballabani/tap/lazyfire</span>
      </div>
      <div class="terminal-line">
        <span class="terminal-prompt">$</span>
        <span class="terminal-cmd">lazyfire</span>
      </div>
    </div>
  </div>
  <div class="quick-start-alt">
    <div class="alt-method">
      <span class="alt-label">Go</span>
      <code>go install github.com/marjoballabani/lazyfire@latest</code>
    </div>
    <div class="alt-method">
      <span class="alt-label">Binary</span>
      <code>curl -L https://github.com/.../lazyfire_darwin_arm64.tar.gz | tar xz</code>
    </div>
    <a :href="withBase('/guide/installation')" class="alt-link">All installation options</a>
  </div>
</div>

<div class="section-divider"></div>

## Preview

<div class="preview-wrapper">
  <div class="preview-frame">
    <div class="terminal-header">
      <div class="terminal-dot red"></div>
      <div class="terminal-dot yellow"></div>
      <div class="terminal-dot green"></div>
      <span class="terminal-title">lazyfire</span>
    </div>
    <img :src="withBase('/preview.gif')" alt="LazyFire Preview" class="preview-gif" />
  </div>
</div>

<div class="section-divider"></div>

## Interface

<div class="interface-demo">
  <div class="panel-layout">
    <div class="panel-left">
      <div class="panel" :class="{ active: true }">
        <div class="panel-title">Projects</div>
        <div class="panel-content">
          <div class="panel-row selected">my-app-prod</div>
          <div class="panel-row">my-app-staging</div>
          <div class="panel-row">my-app-dev</div>
        </div>
        <div class="panel-count">1 of 3</div>
      </div>
      <div class="panel">
        <div class="panel-title">Collections <span class="panel-tab-inactive">Functions</span></div>
        <div class="panel-content">
          <div class="panel-row">Client</div>
          <div class="panel-row">Order</div>
          <div class="panel-row selected">Product</div>
          <div class="panel-row">Retail</div>
        </div>
        <div class="panel-count">3 of 4</div>
      </div>
      <div class="panel">
        <div class="panel-title">Tree</div>
        <div class="panel-content">
          <div class="panel-row selected">9crF4Fg1BpQJt...</div>
          <div class="panel-row">XskAkUIJNmJwy...</div>
          <div class="panel-row">toi6iaX11KC3qW...</div>
        </div>
        <div class="panel-count">1 of 3</div>
      </div>
    </div>
    <div class="panel-right">
      <div class="panel details-panel">
        <div class="panel-title">Details</div>
        <div class="panel-stats">
          <span class="stat green">Size: 369 B / 1MB</span>
          <span class="stat yellow">Index: ~30+</span>
          <span class="stat green">Depth: 1 / 20</span>
        </div>
        <div class="panel-content mono">
          <div class="panel-row">{</div>
          <div class="panel-row">  <span class="json-key">"brand"</span>: <span class="json-str">"Apple"</span>,</div>
          <div class="panel-row">  <span class="json-key">"model"</span>: <span class="json-str">"XS max"</span>,</div>
          <div class="panel-row">  <span class="json-key">"name"</span>: <span class="json-str">"Iphone"</span>,</div>
          <div class="panel-row">  <span class="json-key">"price"</span>: <span class="json-str">"4"</span>,</div>
          <div class="panel-row">  <span class="json-key">"enabled"</span>: <span class="json-bool">true</span>,</div>
          <div class="panel-row">  <span class="json-key">"quantity"</span>: <span class="json-str">"0"</span></div>
          <div class="panel-row">}</div>
        </div>
      </div>
      <div class="panel commands-panel">
        <div class="panel-title">Commands</div>
        <div class="panel-content">
          <div class="panel-row dim">cache Using cached 9crF4Fg1BpQJt...</div>
        </div>
      </div>
    </div>
  </div>
  <div class="panel-statusbar">
    <span class="status-keys"><span class="sk">h/l</span> cols  <span class="sk">j/k</span> move  <span class="sk">space</span> select  <span class="sk">c</span> copy  <span class="sk">/</span> filter  <span class="sk">F</span> query  <span class="sk">?</span> help  <span class="sk">q</span> quit</span>
    <span class="status-version">v0.1.40</span>
  </div>
  <div class="interface-hint">Four-panel layout with vim-style navigation</div>
</div>

<div class="section-divider"></div>

## Why LazyFire?

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
    <div class="why-desc">Uses your existing firebase login credentials. Install and run - that's it.</div>
  </div>
</div>

<div class="section-divider"></div>

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

<div class="section-divider"></div>

<div class="cta-section">
  <h2>Ready to try it?</h2>
  <p class="cta-desc">Install LazyFire and start browsing your Firebase projects in seconds.</p>
  <div class="cta-buttons">
    <a :href="withBase('/guide/getting-started')" class="cta-btn primary">Get Started</a>
    <a href="https://github.com/marjoballabani/lazyfire" class="cta-btn secondary">GitHub</a>
  </div>
</div>

</div>
