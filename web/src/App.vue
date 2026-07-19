<script setup lang="ts"></script>

<template>
  <header class="site-header">
    <RouterLink class="brand" to="/">albatross <span class="brand-accent">leaderboard</span><span class="brand-accent">.</span></RouterLink>
  </header>
  <main>
    <RouterView />
  </main>
  <footer class="site-footer">
    <p class="footer-links">
      <a href="https://github.com/alexbathome/albatross" target="_blank" rel="noopener noreferrer">GitHub</a>
      <span class="footer-sep">·</span>
      <a href="https://discord.gg/sfuM89Sv2G" target="_blank" rel="noopener noreferrer" class="discord-link">
        Submit your scores on Discord
      </a>
    </p>
  </footer>
</template>
