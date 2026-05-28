<template>
  <div class="timing">
    <h1 class="page-title">Race Timing</h1>
    
    <section class="timing-section">
      <h2>Active Events</h2>
      <div class="events-table">
        <div class="table-header">
          <span>Event Name</span>
          <span>Date</span>
          <span>Participants</span>
          <span>Status</span>
        </div>
        <div class="table-body">
          <div class="table-row">
            <span>Sample Active Event</span>
            <span>2024-06-15</span>
            <span>45</span>
            <span class="status-active">Active</span>
          </div>
        </div>
      </div>
    </section>

    <section class="timing-section">
      <h2>Past Events</h2>
      <div class="events-table">
        <div class="table-header">
          <span>Event Name</span>
          <span>Date</span>
          <span>Participants</span>
          <span>Status</span>
        </div>
        <div class="table-body">
          <div class="table-row">
            <span>Sample Completed Event</span>
            <span>2024-05-20</span>
            <span>32</span>
            <span class="status-completed">Completed</span>
          </div>
        </div>
      </div>
    </section>
  </div>
</template>

<script setup>
// Timing component - placeholder for actual implementation
</script>

<style scoped>
.timing {
  max-width: 1200px;
  margin: 0 auto;
  padding: 0 2rem;
}

.page-title {
  text-align: center;
  margin-bottom: 3rem;
  color: #2c3e50;
}

.timing-section {
  margin-bottom: 3rem;
}

.timing-section h2 {
  margin-bottom: 1.5rem;
  color: #2c3e50;
}

.events-table {
  background: white;
  border-radius: 8px;
  box-shadow: 0 2px 10px rgba(0, 0, 0, 0.1);
  overflow: hidden;
}

.table-header {
  display: grid;
  grid-template-columns: 2fr 1fr 1fr 1fr;
  gap: 1rem;
  padding: 1rem;
  background: #f8f9fa;
  font-weight: 600;
  color: #2c3e50;
}

.table-row {
  display: grid;
  grid-template-columns: 2fr 1fr 1fr 1fr;
  gap: 1rem;
  padding: 1rem;
  border-bottom: 1px solid #e9ecef;
  transition: background-color 0.3s;
}

.table-row:hover {
  background: #f8f9fa;
}

.table-row:last-child {
  border-bottom: none;
}

.status-active {
  color: #28a745;
  font-weight: 500;
}

.status-completed {
  color: #6c757d;
  font-weight: 500;
}

@media (max-width: 768px) {
  .timing {
    padding: 0 1rem;
  }
  
  .table-header,
  .table-row {
    grid-template-columns: 1fr;
    gap: 0.5rem;
  }
  
  .table-header span,
  .table-row span {
    text-align: left;
  }
}
</style>