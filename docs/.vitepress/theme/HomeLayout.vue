<script setup>
import DefaultTheme from 'vitepress/theme'
const { Layout } = DefaultTheme
</script>

<template>
  <Layout>
    <template #home-features-before>
      <div class="badges-hero">
        <a href="https://github.com/marjoballabani/lazyfire/releases" class="badge-link">
          <span class="badge-icon">📦</span>
          <span class="badge-label">Release</span>
          <img src="https://img.shields.io/github/v/release/marjoballabani/lazyfire?style=flat-square&color=ff6f00&label=" alt="Release">
        </a>
        <a href="https://github.com/marjoballabani/lazyfire/blob/main/LICENSE" class="badge-link">
          <span class="badge-icon">⚖️</span>
          <span class="badge-label">License</span>
          <img src="https://img.shields.io/github/license/marjoballabani/lazyfire?style=flat-square&label=" alt="License">
        </a>
        <a href="https://github.com/marjoballabani/lazyfire/stargazers" class="badge-link">
          <span class="badge-icon">⭐</span>
          <span class="badge-label">Stars</span>
          <img src="https://img.shields.io/github/stars/marjoballabani/lazyfire?style=flat-square&color=ffca28&label=" alt="Stars">
        </a>
      </div>
    </template>
  </Layout>
</template>

<style>
.badges-hero {
  display: flex;
  justify-content: center;
  gap: 1rem;
  flex-wrap: wrap;
  padding: 2rem 24px 3rem;
}
</style>
