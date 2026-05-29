<template>
  <div id="app">
    <header class="header">
      <nav class="nav">
        <router-link to="/" class="logo">
          <h1>Inferior Timing</h1>
        </router-link>
        <div class="nav-links">
          <router-link to="/" class="nav-link">Home</router-link>
          <router-link to="/timing" class="nav-link">Timing</router-link>
        </div>
      </nav>
    </header>
    
    <main class="main">
      <router-view />
    </main>
    
    <footer class="footer">
      <div class="footer-content">
        <p>&copy; 2024 Keweenaw Endurance. All rights reserved.</p>
      </div>
    </footer>
  </div>
</template>

<script setup lang="ts">
// Main App component
</script>

<style scoped>
.header {
  background-color: #2c3e50;
  color: white;
  padding: 1rem 0;
}

.nav {
  max-width: 1200px;
  margin: 0 auto;
  padding: 0 2rem;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.logo {
  text-decoration: none;
  color: white;
}

.logo h1 {
  margin: 0;
  font-size: 1.5rem;
}

.nav-links {
  display: flex;
  gap: 2rem;
}

.nav-link {
  color: white;
  text-decoration: none;
  padding: 0.5rem 1rem;
  border-radius: 4px;
  transition: background-color 0.3s;
}

.nav-link:hover,
.nav-link.router-link-active {
  background-color: #34495e;
}

.main {
  min-height: calc(100vh - 120px);
  padding: 2rem 0;
}

.footer {
  background-color: #34495e;
  color: white;
  padding: 2rem 0;
  margin-top: auto;
}

.footer-content {
  max-width: 1200px;
  margin: 0 auto;
  padding: 0 2rem;
  text-align: center;
}

@media (max-width: 768px) {
  .nav {
    flex-direction: column;
    gap: 1rem;
  }
  
  .nav-links {
    gap: 1rem;
  }
}
</style>