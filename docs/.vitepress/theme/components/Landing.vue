<script setup lang="ts">
import { ref } from 'vue'
import { withBase } from 'vitepress'
import TuiDemo from './TuiDemo.vue'

const brew = 'brew install marjoballabani/tap/lazyfire'
const copied = ref(false)

async function copyInstall() {
  try {
    await navigator.clipboard.writeText(brew)
    copied.value = true
    setTimeout(() => (copied.value = false), 1800)
  } catch {
    // Clipboard blocked: the command stays selectable
  }
}

// The keys are the structure of this list: each row is what one key does
const groups: { title: string; rows: { keys: string[]; text: string }[] }[] = [
  {
    title: 'Move around',
    rows: [
      { keys: ['tab', 'shift+tab'], text: 'Cycle through Projects, Databases, Collections and Tree.' },
      { keys: ['1', '4'], text: 'Jump straight to a panel. 0 jumps to the document.' },
      { keys: ['?'], text: 'List every key that works where you are, and run one from the list.' },
    ],
  },
  {
    title: 'Browse data',
    rows: [
      { keys: ['space'], text: 'Pick a project or database, open a collection, expand a document.' },
      { keys: ['/'], text: 'Filter any list as you type. In a document, filter lines or run a jq query.' },
      { keys: ['F'], text: 'Build a Firestore query with where clauses, ordering and a limit.' },
    ],
  },
  {
    title: 'Inspect',
    rows: [
      { keys: ['y'], text: 'Copy the value under the cursor. c copies the whole document as JSON.' },
      { keys: ['T'], text: 'Show timestamps as readable dates next to the raw values.' },
      { keys: ['S'], text: 'Check every collection against Firestore size, depth and index limits.' },
    ],
  },
  {
    title: 'Beyond Firestore',
    rows: [
      { keys: ['[', ']'], text: 'Switch to Cloud Functions and their logs, Storage, Auth users, rules and indexes.' },
      { keys: ['L'], text: 'Show only function logs at or above a severity.' },
    ],
  },
]

const comparison: [string, string, string][] = [
  ['Switch projects', 'Click through menus', '1, then j/k and space'],
  ['Switch databases', 'Pick one on every page', '2, then space'],
  ['Open subcollections', 'Click, wait, click again', 'space'],
  ['Check document limits', 'Not available', 'Shown with every document'],
  ['Scan collection health', 'One document at a time', 'S'],
  ['Query documents', 'Web form', 'F'],
  ['Copy a document as JSON', 'Select, copy, reformat', 'c'],
  ['Read function logs', 'Separate Cloud Logging page', '] to the Logs tab'],
]
</script>

<template>
  <div class="landing">
    <header class="hero">
      <h1 class="hero-title">Firebase, from the comfort of your terminal.</h1>
      <p class="hero-lede">
        LazyFire is a keyboard-driven terminal app for Firebase. Browse projects, databases and Firestore documents,
        run queries, and read Cloud Functions logs without opening the console.
      </p>
      <div class="hero-actions">
        <div class="install">
          <code class="install-cmd"><span class="prompt" aria-hidden="true">$ </span>{{ brew }}</code>
          <button type="button" class="install-copy" @click="copyInstall">{{ copied ? 'Copied' : 'Copy' }}</button>
        </div>
        <a class="link-button" :href="withBase('/guide/getting-started')">Read the guide</a>
        <a class="link-plain" href="https://github.com/marjoballabani/lazyfire">GitHub</a>
      </div>
    </header>

    <section class="demo" aria-labelledby="demo-note">
      <TuiDemo />
      <p id="demo-note" class="demo-note">
        This is a working replica with sample data. Click it and try <kbd>tab</kbd> <kbd>j</kbd> <kbd>k</kbd>
        <kbd>space</kbd> <kbd>enter</kbd> <kbd>0</kbd> and <kbd>?</kbd>. <kbd>esc</kbd> gives the page its keys back.
      </p>
    </section>

    <section class="keys" aria-labelledby="keys-title">
      <h2 id="keys-title">Everything is one key away</h2>
      <p class="section-lede">No menus to learn. The bar at the bottom shows the keys for the panel you are in.</p>
      <div class="key-groups">
        <div v-for="group in groups" :key="group.title" class="key-group">
          <h3>{{ group.title }}</h3>
          <dl>
            <template v-for="row in group.rows" :key="row.text">
              <dt>
                <template v-for="(k, i) in row.keys" :key="k">
                  <span v-if="i > 0 && row.keys[0] === '1'" class="key-sep">to</span>
                  <kbd>{{ k }}</kbd>
                </template>
              </dt>
              <dd>{{ row.text }}</dd>
            </template>
          </dl>
        </div>
      </div>
      <a class="section-link" :href="withBase('/reference/keybindings')">All keybindings</a>
    </section>

    <section class="install-section" aria-labelledby="install-title">
      <h2 id="install-title">Install</h2>
      <p class="section-lede">
        LazyFire uses your existing <code>firebase login</code>. There is nothing else to set up.
      </p>
      <dl class="install-list">
        <dt>Homebrew</dt>
        <dd><code>brew install marjoballabani/tap/lazyfire</code></dd>
        <dt>Go</dt>
        <dd><code>go install github.com/marjoballabani/lazyfire@latest</code></dd>
        <dt>Binaries</dt>
        <dd>
          macOS, Linux and Windows builds are on the
          <a href="https://github.com/marjoballabani/lazyfire/releases">releases page</a>.
        </dd>
      </dl>
      <a class="section-link" :href="withBase('/guide/installation')">Installation guide</a>
    </section>

    <section class="compare" aria-labelledby="compare-title">
      <h2 id="compare-title">Firebase Console or LazyFire</h2>
      <table>
        <thead>
          <tr>
            <th scope="col">Task</th>
            <th scope="col">Firebase Console</th>
            <th scope="col">LazyFire</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="[task, console, lazyfire] in comparison" :key="task">
            <th scope="row">{{ task }}</th>
            <td>{{ console }}</td>
            <td class="ours">{{ lazyfire }}</td>
          </tr>
        </tbody>
      </table>
    </section>

    <footer class="closing">
      <p>Open source under the MIT License. Built by Marjo Ballabani.</p>
      <p>
        <a :href="withBase('/guide/getting-started')">Getting started</a>
        <a href="https://github.com/marjoballabani/lazyfire/releases">Releases</a>
        <a href="https://github.com/marjoballabani/lazyfire/blob/main/CHANGELOG.md">Changelog</a>
      </p>
    </footer>
  </div>
</template>

<style scoped>
.landing {
  --measure: 62ch;
  max-width: 1180px;
  margin: 0 auto;
  padding: 0 24px 72px;
}

/* Hero: the headline is set in Recursive's casual style, the rest is quiet */
.hero {
  padding: 64px 0 40px;
}

.hero-title {
  max-width: 16ch;
  margin: 0;
  font-size: clamp(2.5rem, 6.4vw, 4.75rem);
  line-height: 1.02;
  font-weight: 850;
  letter-spacing: -0.025em;
  font-variation-settings: 'MONO' 0, 'CASL' 1;
  color: var(--vp-c-text-1);
}

.hero-lede {
  max-width: var(--measure);
  margin: 24px 0 0;
  font-size: 1.15rem;
  line-height: 1.6;
  color: var(--vp-c-text-2);
}

.hero-actions {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 14px 22px;
  margin-top: 32px;
}

.install {
  display: flex;
  align-items: stretch;
  max-width: 100%;
  border: 1px solid var(--lf-line-dark);
  border-radius: 10px;
  background: var(--lf-surface-dark);
  overflow: hidden;
}

.install-cmd {
  padding: 11px 14px;
  background: none;
  color: var(--lf-text-dark);
  font-size: 0.92rem;
  white-space: nowrap;
  overflow-x: auto;
}

.prompt {
  color: var(--lf-ember);
  user-select: none;
}

.install-copy {
  padding: 0 16px;
  border-left: 1px solid var(--lf-line-dark);
  color: var(--lf-flame);
  font-size: 0.9rem;
  font-weight: 600;
}

.install-copy:hover {
  background: rgba(255, 209, 102, 0.08);
}

.link-button {
  padding: 10px 18px;
  border-radius: 10px;
  background: var(--vp-button-brand-bg);
  color: var(--vp-button-brand-text);
  font-weight: 650;
  text-decoration: none;
}

.link-button:hover {
  background: var(--vp-button-brand-hover-bg);
}

.link-plain {
  color: var(--vp-c-text-1);
  font-weight: 600;
  text-decoration: underline;
  text-underline-offset: 4px;
  text-decoration-color: var(--vp-c-border);
}

.link-plain:hover {
  text-decoration-color: currentColor;
}

/* Demo */
.demo {
  margin: 8px 0 0;
}

.demo-note {
  max-width: 80ch;
  margin: 16px 0 0;
  color: var(--vp-c-text-2);
  font-size: 0.95rem;
  line-height: 1.8;
}

/* Sections share one rhythm */
.keys,
.install-section,
.compare {
  margin-top: 104px;
}

h2 {
  margin: 0;
  font-size: clamp(1.7rem, 3.4vw, 2.3rem);
  line-height: 1.15;
  font-weight: 800;
  letter-spacing: -0.015em;
  font-variation-settings: 'MONO' 0, 'CASL' 0.8;
  color: var(--vp-c-text-1);
}

.section-lede {
  max-width: var(--measure);
  margin: 12px 0 0;
  color: var(--vp-c-text-2);
  line-height: 1.6;
}

.section-lede code,
.install-list code {
  font-size: 0.9em;
  padding: 2px 6px;
  border-radius: 5px;
  background: var(--vp-code-bg);
  color: var(--vp-code-color);
}

.section-link {
  display: inline-block;
  margin-top: 28px;
  color: var(--vp-c-brand-1);
  font-weight: 600;
  text-underline-offset: 4px;
}

/* Keys list: keycaps on the left carry the meaning */
.key-groups {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 44px 64px;
  margin-top: 40px;
}

.key-group h3 {
  margin: 0 0 14px;
  font-size: 1.05rem;
  font-weight: 700;
  font-variation-settings: 'MONO' 0, 'CASL' 0.5;
  color: var(--vp-c-text-1);
}

.key-group dl {
  display: grid;
  grid-template-columns: 8.5rem 1fr;
  gap: 14px 18px;
  margin: 0;
}

.key-group dt {
  display: flex;
  flex-wrap: wrap;
  align-items: baseline;
  gap: 6px;
}

.key-group dd {
  margin: 0;
  color: var(--vp-c-text-2);
  line-height: 1.55;
}

.key-sep {
  color: var(--vp-c-text-3);
  font-size: 0.85rem;
}

/* Install */
.install-list {
  display: grid;
  grid-template-columns: 7rem 1fr;
  gap: 16px 20px;
  margin: 32px 0 0;
  max-width: 760px;
}

.install-list dt {
  font-weight: 700;
  color: var(--vp-c-text-1);
}

.install-list dd {
  margin: 0;
  color: var(--vp-c-text-2);
}

.install-list a {
  color: var(--vp-c-brand-1);
}

/* Comparison */
.compare table {
  width: 100%;
  max-width: 900px;
  margin-top: 32px;
  border-collapse: collapse;
}

.compare th,
.compare td {
  padding: 12px 16px 12px 0;
  text-align: left;
  vertical-align: top;
  border-bottom: 1px solid var(--vp-c-divider);
}

.compare thead th {
  color: var(--vp-c-text-3);
  font-weight: 600;
  font-size: 0.9rem;
}

.compare tbody th {
  font-weight: 600;
  color: var(--vp-c-text-1);
}

.compare td {
  color: var(--vp-c-text-2);
}

.compare td.ours {
  color: var(--vp-c-text-1);
  font-family: var(--vp-font-family-mono);
  font-variation-settings: 'MONO' 1, 'CASL' 0;
  font-size: 0.92rem;
}

/* Closing */
.closing {
  margin-top: 104px;
  padding-top: 28px;
  border-top: 1px solid var(--vp-c-divider);
  color: var(--vp-c-text-3);
  font-size: 0.92rem;
}

.closing p {
  margin: 0 0 8px;
}

.closing a {
  margin-right: 20px;
  color: var(--vp-c-text-2);
}

.closing a:hover {
  color: var(--vp-c-brand-1);
}

@media (max-width: 860px) {
  .key-groups {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 560px) {
  .hero {
    padding-top: 40px;
  }

  .key-group dl,
  .install-list {
    grid-template-columns: 1fr;
    gap: 6px;
  }

  .key-group dd,
  .install-list dd {
    margin-bottom: 12px;
  }

  .install-cmd {
    font-size: 0.8rem;
  }

  .compare table,
  .compare thead,
  .compare tbody,
  .compare tr,
  .compare th,
  .compare td {
    display: block;
  }

  .compare thead {
    display: none;
  }

  .compare tr {
    padding: 12px 0;
    border-bottom: 1px solid var(--vp-c-divider);
  }

  .compare th,
  .compare td {
    padding: 2px 0;
    border: none;
  }
}
</style>
