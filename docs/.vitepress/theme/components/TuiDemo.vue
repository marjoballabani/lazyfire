<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref } from 'vue'

// A small working replica of LazyFire with sample data. It follows the real
// keybindings, so visitors learn the app by using it.

type Panel = 'projects' | 'databases' | 'collections' | 'tree' | 'details'
type Seg = { t: string; c?: string }

const sidePanels: Panel[] = ['projects', 'databases', 'collections', 'tree']
const panelNames: Record<Panel, string> = {
  projects: 'Projects',
  databases: 'Databases',
  collections: 'Collections',
  tree: 'Tree',
  details: 'Details',
}

const projects = [
  { id: 'shop-prod', name: 'Shop (prod)', databases: ['(default)', 'orders-eu'] },
  { id: 'shop-staging', name: 'Shop (staging)', databases: ['(default)'] },
  { id: 'analytics-dev', name: 'Analytics (dev)', databases: ['(default)'] },
]
const databaseLocations: Record<string, string> = { '(default)': 'nam5', 'orders-eu': 'eur3' }
const collectionsByDatabase: Record<string, string[]> = {
  '(default)': ['users', 'products', 'reviews', 'carts'],
  'orders-eu': ['orders', 'invoices', 'refunds'],
}
const documents: Record<string, { id: string; data: Record<string, unknown> }[]> = {
  users: [
    { id: 'u_alice', data: { name: 'Alice Moreau', email: 'alice@example.com', plan: 'pro', seats: 5, active: true, createdAt: '2025-11-02T09:14:00Z' } },
    { id: 'u_bruno', data: { name: 'Bruno Costa', email: 'bruno@example.com', plan: 'free', seats: 1, active: true, createdAt: '2026-01-19T17:40:00Z' } },
    { id: 'u_chen', data: { name: 'Chen Wei', email: 'chen@example.com', plan: 'team', seats: 12, active: false, createdAt: '2024-06-08T08:03:00Z' } },
  ],
  products: [
    { id: 'p_lamp', data: { title: 'Ember desk lamp', price: 49.0, stock: 132, tags: ['lighting', 'desk'] } },
    { id: 'p_mug', data: { title: 'Night owl mug', price: 14.5, stock: 0, tags: ['kitchen'] } },
  ],
  reviews: [{ id: 'r_0913', data: { product: 'p_lamp', rating: 5, text: 'Warm light, no glare.' } }],
  carts: [{ id: 'c_77ab', data: { user: 'u_bruno', items: 2, total: 63.5 } }],
  orders: [
    { id: 'ord_1042', data: { customer: 'u_alice', status: 'shipped', total: 129.5, currency: 'EUR', items: 3, placedAt: '2026-09-18T10:22:00Z' } },
    { id: 'ord_1043', data: { customer: 'u_chen', status: 'paid', total: 64.0, currency: 'EUR', items: 1, placedAt: '2026-09-20T21:05:00Z' } },
    { id: 'ord_1044', data: { customer: 'u_bruno', status: 'refunded', total: 14.5, currency: 'EUR', items: 1, placedAt: '2026-09-21T07:48:00Z' } },
  ],
  invoices: [{ id: 'inv_2231', data: { order: 'ord_1042', amount: 129.5, paid: true } }],
  refunds: [{ id: 'rf_017', data: { order: 'ord_1044', amount: 14.5, reason: 'damaged' } }],
}

const initialState = () => ({
  focus: 'projects' as Panel,
  previous: 'projects' as Panel,
  project: 0,
  database: '(default)',
  collection: null as string | null,
  docPath: null as string | null,
  sel: { projects: 0, databases: 0, collections: 0, tree: 0 } as Record<string, number>,
  cursor: 0,
  menuOpen: false,
  command: 'auth Using local Firebase/gcloud authentication',
})
const state = reactive(initialState())

const root = ref<HTMLElement | null>(null)
const pressed = ref('')
const toast = ref('')
const announcement = ref('')

const databases = computed(() => projects[state.project].databases)
const collections = computed(() => collectionsByDatabase[state.database])
const treeDocs = computed(() => (state.collection ? documents[state.collection] : []))
const openDoc = computed(() => {
  if (!state.docPath) return null
  const [collection, id] = state.docPath.split('/')
  return documents[collection]?.find((d) => d.id === id) ?? null
})

function listLength(panel: Panel): number {
  switch (panel) {
    case 'projects':
      return projects.length
    case 'databases':
      return databases.value.length
    case 'collections':
      return collections.value.length
    case 'tree':
      return treeDocs.value.length
    default:
      return detailsLines.value.length
  }
}

// Details content, drawn like the real panel: header, stats, JSON
function jsonLines(data: Record<string, unknown>): Seg[][] {
  const lines: Seg[][] = [[{ t: '{' }]]
  const entries = Object.entries(data)
  entries.forEach(([key, value], i) => {
    const comma = i < entries.length - 1 ? ',' : ''
    let v: Seg
    if (typeof value === 'string') v = { t: JSON.stringify(value), c: 'str' }
    else if (typeof value === 'number') v = { t: String(value), c: 'num' }
    else if (typeof value === 'boolean') v = { t: String(value), c: 'bool' }
    else v = { t: JSON.stringify(value), c: 'str' }
    lines.push([{ t: '  ' }, { t: JSON.stringify(key), c: 'key' }, { t: ': ' }, v, { t: comma }])
  })
  lines.push([{ t: '}' }])
  return lines
}

const detailsLines = computed<Seg[][]>(() => {
  const doc = openDoc.value
  if (doc) {
    const size = JSON.stringify(doc.data).length + (state.docPath ?? '').length + 48
    const fields = Object.keys(doc.data).length
    const schema = Object.entries(doc.data)
      .map(([k, v]) => `${k}:${Array.isArray(v) ? 'array' : typeof v === 'number' ? (Number.isInteger(v) ? 'int' : 'float') : typeof v === 'boolean' ? 'bool' : 'string'}`)
      .sort()
      .join(', ')
    return [
      [{ t: `─── ${state.docPath} ───`, c: 'rule' }],
      [{ t: 'Size: ', c: 'dim' }, { t: `${size} B / 1MB`, c: 'ok' }, { t: '  Index Entries: ', c: 'dim' }, { t: `${fields * 2} / 40000`, c: 'ok' }, { t: '  Depth: ', c: 'dim' }, { t: '1 / 20', c: 'ok' }],
      [{ t: `Schema: ${schema}`, c: 'dim' }],
      [{ t: '' }],
      ...jsonLines(doc.data),
    ]
  }
  switch (state.focus) {
    case 'projects': {
      const p = projects[state.sel.projects]
      return [[{ t: '─── Project Info ───', c: 'rule' }], [{ t: '' }], [{ t: '  ID:    ', c: 'label' }, { t: p.id }], [{ t: '  Name:  ', c: 'label' }, { t: p.name }], [{ t: '' }], [{ t: '  Press Space to select project', c: 'dim' }]]
    }
    case 'databases': {
      const id = databases.value[state.sel.databases]
      return [[{ t: '─── Database Info ───', c: 'rule' }], [{ t: '' }], [{ t: '  ID:        ', c: 'label' }, { t: id }], [{ t: '  Location:  ', c: 'label' }, { t: databaseLocations[id] }], [{ t: '  Type:      ', c: 'label' }, { t: 'FIRESTORE_NATIVE' }], [{ t: '' }], [{ t: '  Press Space to use this database', c: 'dim' }]]
    }
    case 'collections': {
      const name = collections.value[state.sel.collections]
      return [[{ t: '─── Collection Info ───', c: 'rule' }], [{ t: '' }], [{ t: '  Name:  ', c: 'label' }, { t: name }], [{ t: '  Path:  ', c: 'label' }, { t: `/${name}` }], [{ t: '' }], [{ t: '  Press Space to browse documents', c: 'dim' }]]
    }
    case 'tree':
      if (!treeDocs.value.length) {
        return [[{ t: '─── Tree ───', c: 'rule' }], [{ t: '' }], [{ t: '  No documents loaded', c: 'dim' }], [{ t: '' }], [{ t: '  Select a collection first', c: 'dim' }]]
      }
      return [[{ t: '─── Node Info ───', c: 'rule' }], [{ t: '' }], [{ t: '  Name:  ', c: 'label' }, { t: treeDocs.value[state.sel.tree].id }], [{ t: '  Type:  ', c: 'label' }, { t: 'document' }], [{ t: '' }], [{ t: '  Press Enter to open it', c: 'dim' }]]
    default:
      return [[{ t: '' }], [{ t: '  L A Z Y F I R E', c: 'rule' }], [{ t: '' }], [{ t: '  Select a project to start', c: 'dim' }]]
  }
})

// Bottom bar hints follow the focused panel, like the real app
const hints = computed<[string, string][]>(() => {
  if (state.menuOpen) return [['esc', 'close'], ['enter', 'execute'], ['/', 'filter']]
  const common: [string, string][] = [['j/k', 'move'], ['tab', 'panels']]
  switch (state.focus) {
    case 'projects':
      return [['space', 'select'], ['S', 'scan'], ['/', 'filter'], ['r', 'refresh'], ...common, ['?', 'help']]
    case 'databases':
      return [['space', 'select'], ['/', 'filter'], ['r', 'refresh'], ...common, ['?', 'help']]
    case 'collections':
      return [['space', 'open'], ['F', 'query'], ['/', 'filter'], ['r', 'refresh'], ...common, ['[/]', 'tabs'], ['?', 'help']]
    case 'tree':
      return [['space', 'expand'], ['enter', 'open'], ['v', 'select'], ['/', 'filter'], ['F', 'query'], ['c', 'copy'], ...common, ['?', 'help']]
    default:
      return [['esc', 'back'], ['/', 'filter'], ['c', 'copy'], ['y', 'copy value'], ['t', 'compact'], ['w', 'wrap'], ['j/k', 'move'], ['?', 'help']]
  }
})

const breadcrumb = computed(() => {
  const parts = [projects[state.project].id]
  if (state.database !== '(default)') parts.push(state.database)
  if (state.collection) parts.push(state.collection)
  if (state.docPath) parts.push(state.docPath.split('/')[1])
  return parts.join(' > ')
})

const menuSections = computed(() => {
  const local: Record<Panel, [string, string][]> = {
    projects: [['space', 'Select project'], ['enter', 'Show project details'], ['S', 'Scan collections health']],
    databases: [['space', 'Use database'], ['enter', 'Use database and focus collections']],
    collections: [['space', 'Open collection'], ['enter', 'Open collection and focus tree'], ['F', 'Query builder']],
    tree: [['space', 'Expand / collapse'], ['enter', 'Open document'], ['v', 'Toggle select mode'], ['c', 'Copy JSON to clipboard']],
    details: [['esc', 'Back to previous panel'], ['y', 'Copy value on cursor line'], ['t', 'Toggle compact JSON'], ['w', 'Toggle word wrap']],
  }
  return [
    { title: panelNames[state.focus], rows: local[state.focus] },
    { title: 'Navigation', rows: [['j/k', 'Move down / up'], ['tab', 'Next panel'], ['shift+tab', 'Previous panel'], ['0', 'Focus details'], ['1-4', 'Focus a left panel']] as [string, string][] },
  ]
})

// Actions

function focusPanel(panel: Panel) {
  if (panel === 'details' && state.focus !== 'details') state.previous = state.focus
  state.focus = panel
  announcement.value = `${panelNames[panel]} panel`
}

function move(delta: number) {
  if (state.focus === 'details') {
    state.cursor = Math.max(0, Math.min(state.cursor + delta, detailsLines.value.length - 1))
    return
  }
  const n = listLength(state.focus)
  if (!n) return
  state.sel[state.focus] = Math.max(0, Math.min(state.sel[state.focus] + delta, n - 1))
}

function cycle(dir: number) {
  const i = sidePanels.indexOf(state.focus)
  focusPanel(sidePanels[(i + dir + sidePanels.length) % sidePanels.length])
}

function back() {
  focusPanel(state.previous === 'details' ? 'tree' : state.previous)
}

function select(): boolean {
  switch (state.focus) {
    case 'projects':
      state.project = state.sel.projects
      state.database = '(default)'
      state.sel.databases = 0
      state.collection = null
      state.docPath = null
      state.sel.collections = 0
      state.command = `api ListCollections(${projects[state.project].id}) → ${collections.value.length} collections`
      return true
    case 'databases':
      state.database = databases.value[state.sel.databases]
      state.collection = null
      state.docPath = null
      state.sel.collections = 0
      state.command = `database Using database ${state.database}`
      return true
    case 'collections':
      state.collection = collections.value[state.sel.collections]
      state.sel.tree = 0
      state.command = `api ListDocuments(${state.collection}) → ${treeDocs.value.length} docs`
      return true
    case 'tree': {
      const doc = treeDocs.value[state.sel.tree]
      if (!doc) return false
      state.docPath = `${state.collection}/${doc.id}`
      state.cursor = 0
      state.command = `cache Using cached ${doc.id}`
      return true
    }
  }
  return false
}

function enter() {
  switch (state.focus) {
    case 'databases':
      select()
      focusPanel('collections')
      break
    case 'collections':
      select()
      focusPanel('tree')
      break
    case 'tree':
      if (select()) focusPanel('details')
      break
  }
}

let toastTimer: ReturnType<typeof setTimeout> | undefined
function showToast(text: string) {
  toast.value = text
  clearTimeout(toastTimer)
  toastTimer = setTimeout(() => (toast.value = ''), 2600)
}

const appOnlyKeys = new Set('/FcspytwTSvrx@[]LBDHenNCQAMiR'.split(''))

// Returns false for keys the demo leaves to the page
function handleKey(key: string): boolean {
  pressed.value = key === ' ' ? 'space' : key
  if (state.menuOpen) {
    if (key === '?' || key === 'Escape' || key === 'q') state.menuOpen = false
    return true
  }
  switch (key) {
    case '?':
      state.menuOpen = true
      return true
    case '0':
      focusPanel('details')
      return true
    case '1':
    case '2':
    case '3':
    case '4':
      focusPanel(sidePanels[Number(key) - 1])
      return true
    case 'Tab':
    case 'l':
    case 'ArrowRight':
      state.focus === 'details' ? back() : cycle(1)
      return true
    case 'shift+Tab':
    case 'h':
    case 'ArrowLeft':
      state.focus === 'details' ? back() : cycle(-1)
      return true
    case 'j':
    case 'ArrowDown':
      move(1)
      return true
    case 'k':
    case 'ArrowUp':
      move(-1)
      return true
    case 'g':
      move(-1000)
      return true
    case 'G':
      move(1000)
      return true
    case ' ':
      select()
      return true
    case 'Enter':
      enter()
      return true
    case 'Escape':
      if (state.focus === 'details') {
        back()
        return true
      }
      // Nothing to go back to: hand the keyboard back to the page
      root.value?.blur()
      return true
    case 'q':
      showToast('q quits the real app')
      return true
  }
  if (appOnlyKeys.has(key)) {
    showToast(`${key} works in the real app`)
    return true
  }
  return false
}

function onKeydown(e: KeyboardEvent) {
  if (e.metaKey || e.ctrlKey || e.altKey) return
  stopAutoplay()
  const key = e.key === 'Tab' && e.shiftKey ? 'shift+Tab' : e.key
  if (handleKey(key)) e.preventDefault()
}

function onRowClick(panel: Panel, index: number, e: MouseEvent) {
  stopAutoplay()
  if (state.focus !== panel) focusPanel(panel)
  if (panel === 'details') state.cursor = index
  else state.sel[panel] = index
  if (e.detail === 2) enter()
}

function onPaneClick(panel: Panel) {
  stopAutoplay()
  if (state.focus !== panel) focusPanel(panel)
}

// A short demo plays until the visitor takes over

const script: [string, number][] = [
  ['Tab', 1300],
  ['j', 900],
  ['Enter', 1300],
  ['Enter', 1300],
  ['j', 900],
  ['Enter', 1400],
  ['j', 600],
  ['j', 600],
  ['j', 900],
  ['?', 2200],
  ['?', 900],
  ['Escape', 2600],
]
let step = 0
let timer: ReturnType<typeof setTimeout> | undefined
let autoplay = false
let observer: IntersectionObserver | undefined

function tick() {
  if (!autoplay) return
  if (step >= script.length) {
    Object.assign(state, initialState())
    pressed.value = ''
    step = 0
    timer = setTimeout(tick, 1600)
    return
  }
  const [key, delay] = script[step++]
  handleKey(key)
  timer = setTimeout(tick, delay)
}

function stopAutoplay() {
  autoplay = false
  clearTimeout(timer)
  observer?.disconnect()
}

onMounted(() => {
  if (window.matchMedia('(prefers-reduced-motion: reduce)').matches || !root.value) return
  autoplay = true
  observer = new IntersectionObserver(([entry]) => {
    clearTimeout(timer)
    if (autoplay && entry.isIntersecting) timer = setTimeout(tick, 1200)
  })
  observer.observe(root.value)
})

onBeforeUnmount(() => {
  stopAutoplay()
  clearTimeout(toastTimer)
})

function footer(panel: Panel): string {
  const n = listLength(panel)
  return n ? `${Math.min(state.sel[panel], n - 1) + 1} of ${n}` : '0 of 0'
}
</script>

<template>
  <div class="tui-frame">
    <div class="tui-bar">
      <span class="tui-name">lazyfire</span>
      <span class="tui-pressed" aria-hidden="true">
        <kbd v-if="pressed" :key="pressed + state.focus + state.sel[state.focus]">{{ pressed === 'shift+Tab' ? 'shift+tab' : pressed }}</kbd>
      </span>
    </div>

    <div
      ref="root"
      class="tui"
      tabindex="0"
      role="application"
      aria-roledescription="LazyFire demo"
      aria-label="Interactive LazyFire demo with sample data. Tab and Shift+Tab switch panels, j and k move, Space selects, Enter opens, 0 focuses details, question mark lists keys, Escape leaves the demo."
      @keydown="onKeydown"
      @pointerdown="stopAutoplay"
    >
      <div class="tui-screen">
        <div class="tui-left">
          <section class="pane" :class="{ active: state.focus === 'projects', collapsed: state.focus !== 'projects' }" @click="onPaneClick('projects')">
            <h3 class="pane-title">Projects</h3>
            <ol v-if="state.focus === 'projects'" class="rows">
              <li v-for="(p, i) in projects" :key="p.id" class="row" :class="{ sel: state.sel.projects === i }" @click.stop="onRowClick('projects', i, $event)">
                <span class="mark">{{ i === state.project ? '*' : ' ' }}</span> {{ p.name }}
              </li>
            </ol>
            <div v-else class="row"><span class="mark">*</span> {{ projects[state.project].name }}</div>
            <span v-if="state.focus === 'projects'" class="pane-count">{{ footer('projects') }}</span>
          </section>

          <section class="pane" :class="{ active: state.focus === 'databases', collapsed: state.focus !== 'databases' }" @click="onPaneClick('databases')">
            <h3 class="pane-title">Databases</h3>
            <ol v-if="state.focus === 'databases'" class="rows">
              <li v-for="(d, i) in databases" :key="d" class="row" :class="{ sel: state.sel.databases === i }" @click.stop="onRowClick('databases', i, $event)">
                <span class="mark">{{ d === state.database ? '*' : ' ' }}</span> {{ d }} <span class="dim">({{ databaseLocations[d] }})</span>
              </li>
            </ol>
            <div v-else class="row"><span class="mark">*</span> {{ state.database }}</div>
            <span v-if="state.focus === 'databases'" class="pane-count">{{ footer('databases') }}</span>
          </section>

          <section class="pane grow" :class="{ active: state.focus === 'collections', big: state.focus === 'collections' }" @click="onPaneClick('collections')">
            <h3 class="pane-title"><span class="tab-on">Collections</span> - Functions - Storage &gt;</h3>
            <ol class="rows">
              <li v-for="(c, i) in collections" :key="c" class="row" :class="{ sel: state.focus === 'collections' && state.sel.collections === i }" @click.stop="onRowClick('collections', i, $event)">
                <span class="mark">{{ c === state.collection ? '*' : ' ' }}</span> {{ c }}
              </li>
            </ol>
            <span class="pane-count">{{ footer('collections') }}</span>
          </section>

          <section class="pane grow" :class="{ active: state.focus === 'tree', big: state.focus === 'tree' }" @click="onPaneClick('tree')">
            <h3 class="pane-title">Tree</h3>
            <ol class="rows">
              <li v-for="(d, i) in treeDocs" :key="d.id" class="row" :class="{ sel: state.focus === 'tree' && state.sel.tree === i }" @click.stop="onRowClick('tree', i, $event)">
                <span class="mark">{{ `${state.collection}/${d.id}` === state.docPath ? '*' : ' ' }}</span> {{ d.id }}
              </li>
            </ol>
            <span class="pane-count">{{ footer('tree') }}</span>
          </section>
        </div>

        <div class="tui-right">
          <section class="pane grow" :class="{ active: state.focus === 'details' }" @click="onPaneClick('details')">
            <h3 class="pane-title">Details</h3>
            <div class="details">
              <div
                v-for="(line, i) in detailsLines"
                :key="i"
                class="line"
                :class="{ sel: state.focus === 'details' && state.cursor === i }"
                @click.stop="onRowClick('details', i, $event)"
              ><span v-for="(seg, j) in line" :key="j" :class="seg.c">{{ seg.t }}</span>&#8203;</div>
            </div>
          </section>
          <section class="pane commands">
            <h3 class="pane-title">Commands</h3>
            <div class="row"><span class="ok">✓</span> {{ state.command }}</div>
          </section>
        </div>
      </div>

      <div class="tui-options">
        <span class="hints">
          <span v-for="[k, label] in hints" :key="k + label" class="hint"><span class="hk">{{ k }}</span> {{ label }}</span>
        </span>
        <span v-if="toast" class="toast">{{ toast }}</span>
        <span v-else class="crumb">{{ breadcrumb }}</span>
      </div>

      <div v-if="state.menuOpen" class="menu" role="dialog" aria-label="Keybindings">
        <h3 class="pane-title">Keybindings</h3>
        <template v-for="section in menuSections" :key="section.title">
          <div class="menu-section">─── {{ section.title }} ───</div>
          <div v-for="[k, label] in section.rows" :key="k" class="menu-row"><span class="mk">{{ k }}</span>{{ label }}</div>
        </template>
      </div>

      <span class="sr-only" aria-live="polite">{{ announcement }}</span>
    </div>
  </div>
</template>

<style scoped>
.tui-frame {
  --term: #1c1d20;
  --term-line: #3a3d43;
  --term-text: #d0d2d6;
  --term-dim: #8b8e94;
  --term-sel: #34373e;
  --ember: #ff8a3d;
  --flame: #ffd166;
  --coal: #ff6b57;
  --sky: #8ab4f8;
  --ok: #7ed9a0;

  border: 1px solid var(--term-line);
  border-radius: 12px;
  background: var(--term);
  color: var(--term-text);
  box-shadow: 0 30px 80px -30px rgba(0, 0, 0, 0.6), 0 0 0 1px rgba(255, 138, 61, 0.06);
  overflow: hidden;
}

.tui-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0.45rem 0.9rem;
  border-bottom: 1px solid var(--term-line);
  font-family: var(--vp-font-family-mono);
  font-variation-settings: 'MONO' 1, 'CASL' 0;
  font-size: 12px;
  color: var(--term-dim);
}

.tui-pressed kbd {
  background: #2a2c31;
  border-color: #4a4d55;
  color: var(--flame);
  animation: press 380ms ease-out;
}

@keyframes press {
  from {
    transform: translateY(2px);
    border-bottom-width: 1px;
  }
}

.tui {
  position: relative;
  outline: none;
  overflow-x: auto;
  font-family: var(--vp-font-family-mono);
  font-variation-settings: 'MONO' 1, 'CASL' 0;
  font-size: 13px;
  line-height: 1.55;
  cursor: default;
}

.tui:focus-visible {
  box-shadow: inset 0 0 0 2px var(--ember);
}

.tui-screen {
  display: grid;
  grid-template-columns: minmax(230px, 1fr) 2fr;
  gap: 0 4px;
  min-width: 720px;
  height: 430px;
  padding: 12px 10px 4px;
}

.tui-left,
.tui-right {
  display: flex;
  flex-direction: column;
  gap: 18px;
  min-height: 0;
}

.pane {
  position: relative;
  display: flex;
  flex-direction: column;
  border: 1px solid var(--term-line);
  border-radius: 7px;
  padding: 0.55em 0.5em 0.45em;
  min-height: 0;
}

.pane.collapsed {
  flex: 0 0 auto;
}

.pane.grow {
  flex: 1 1 0;
}

/* Titles sit on the border, so only the content clips */
.pane.grow .rows,
.pane.grow .details {
  flex: 1 1 0;
  min-height: 0;
  overflow: hidden;
}

.pane.big {
  flex-grow: 2;
}

.pane.active {
  border-color: var(--ember);
}

.pane-title {
  position: absolute;
  top: -0.8em;
  left: 0.7em;
  margin: 0;
  padding: 0 0.35em;
  background: var(--term);
  font: inherit;
  font-weight: 400;
  color: var(--term-dim);
  white-space: nowrap;
}

.pane.active .pane-title {
  color: var(--ember);
}

.tab-on {
  color: var(--ember);
}

.pane-count {
  position: absolute;
  right: 0.7em;
  bottom: -0.8em;
  padding: 0 0.35em;
  background: var(--term);
  color: var(--term-dim);
}

.pane.active .pane-count {
  color: var(--ember);
}

.rows {
  list-style: none;
  margin: 0;
  padding: 0;
}

.row,
.line {
  padding: 0 0.3em;
  white-space: pre;
  overflow: hidden;
  text-overflow: ellipsis;
  border-radius: 3px;
}

.row.sel,
.line.sel {
  background: var(--term-sel);
  color: #fff;
}

.mark {
  color: var(--ember);
}

.dim {
  color: var(--term-dim);
}

.key {
  color: var(--sky);
}

.str {
  color: var(--flame);
}

.num {
  color: var(--ember);
}

.bool {
  color: var(--coal);
}

.rule {
  color: #7fd3e6;
}

.label {
  color: var(--flame);
}

.ok {
  color: var(--ok);
}

.commands {
  flex: 0 0 auto;
}

.tui-options {
  display: flex;
  justify-content: space-between;
  gap: 1rem;
  min-width: 720px;
  padding: 0.35rem 1rem 0.6rem;
  white-space: nowrap;
}

.hints {
  overflow: hidden;
  text-overflow: ellipsis;
}

.hint + .hint {
  margin-left: 1.1em;
}

.hk {
  color: var(--ember);
}

.crumb {
  color: var(--term-dim);
}

.toast {
  color: var(--coal);
}

.menu {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -54%);
  width: min(430px, 80%);
  padding: 0.9em 0.8em 0.7em;
  border: 1px solid var(--ember);
  border-radius: 7px;
  background: var(--term);
  box-shadow: 0 18px 50px rgba(0, 0, 0, 0.55);
}

.menu-section {
  color: #7fd3e6;
  margin-top: 0.2em;
}

.menu-row {
  padding-left: 0.6em;
}

.mk {
  display: inline-block;
  width: 7.5em;
  color: var(--flame);
}

.sr-only {
  position: absolute;
  width: 1px;
  height: 1px;
  overflow: hidden;
  clip: rect(0 0 0 0);
  white-space: nowrap;
}

@media (prefers-reduced-motion: reduce) {
  .tui-pressed kbd {
    animation: none;
  }
}
</style>
