<script setup>
import { ref } from 'vue'
import { useAuthStore } from '@/store/auth'

const authStore = useAuthStore()

const settings = ref({
  notifications: true,
  emailAlerts: false,
  darkMode: true
})
</script>

<template>
  <div class="settings-page">
    <div class="page-header">
      <h2>Settings</h2>
      <p>Manage your account and preferences</p>
    </div>

    <div class="settings-sections">
      <section class="settings-section card">
        <h3>Profile</h3>
        <div class="profile-info">
          <div class="avatar-large">
            {{ authStore.userName?.charAt(0)?.toUpperCase() || 'U' }}
          </div>
          <div class="profile-details">
            <div class="form-group">
              <label class="form-label">Name</label>
              <input type="text" class="form-input" :value="authStore.userName" />
            </div>
            <div class="form-group">
              <label class="form-label">Email</label>
              <input type="email" class="form-input" :value="authStore.user?.email" disabled />
            </div>
          </div>
        </div>
        <button class="btn btn--primary">Save Changes</button>
      </section>

      <section class="settings-section card">
        <h3>Preferences</h3>
        <div class="settings-list">
          <label class="setting-item">
            <div class="setting-info">
              <span class="setting-name">Push Notifications</span>
              <span class="setting-desc">Receive notifications for job completions</span>
            </div>
            <input type="checkbox" v-model="settings.notifications" class="toggle" />
          </label>
          <label class="setting-item">
            <div class="setting-info">
              <span class="setting-name">Email Alerts</span>
              <span class="setting-desc">Get email updates for important events</span>
            </div>
            <input type="checkbox" v-model="settings.emailAlerts" class="toggle" />
          </label>
        </div>
      </section>

      <section class="settings-section card">
        <h3>Security</h3>
        <div class="form-group">
          <label class="form-label">Current Password</label>
          <input type="password" class="form-input" placeholder="Enter current password" />
        </div>
        <div class="form-group">
          <label class="form-label">New Password</label>
          <input type="password" class="form-input" placeholder="Enter new password" />
        </div>
        <div class="form-group">
          <label class="form-label">Confirm Password</label>
          <input type="password" class="form-input" placeholder="Confirm new password" />
        </div>
        <button class="btn btn--primary">Update Password</button>
      </section>

      <section class="settings-section card danger-zone">
        <h3>Danger Zone</h3>
        <p>Permanently delete your account and all associated data.</p>
        <button class="btn btn--danger">Delete Account</button>
      </section>
    </div>
  </div>
</template>

<style lang="scss" scoped>
.settings-page {
  animation: fadeIn 0.3s ease-out;
  max-width: 800px;
}

.page-header {
  margin-bottom: 2rem;
  
  h2 {
    margin-bottom: 0.25rem;
  }
  
  p {
    color: var(--text-muted);
  }
}

.settings-sections {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

.settings-section {
  h3 {
    margin-bottom: 1.5rem;
    padding-bottom: 1rem;
    border-bottom: 1px solid var(--border-color);
  }
}

.profile-info {
  display: flex;
  gap: 2rem;
  margin-bottom: 1.5rem;
}

.avatar-large {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 80px;
  height: 80px;
  background: var(--gradient-primary);
  border-radius: var(--radius-lg);
  font-size: 2rem;
  font-weight: 600;
  color: white;
  flex-shrink: 0;
}

.profile-details {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.settings-list {
  display: flex;
  flex-direction: column;
}

.setting-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 1rem 0;
  border-bottom: 1px solid var(--border-color);
  cursor: pointer;
  
  &:last-child {
    border-bottom: none;
  }
}

.setting-info {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.setting-name {
  font-weight: 500;
  color: var(--text-primary);
}

.setting-desc {
  font-size: 0.875rem;
  color: var(--text-muted);
}

.toggle {
  position: relative;
  width: 44px;
  height: 24px;
  appearance: none;
  background: var(--bg-tertiary);
  border-radius: 12px;
  cursor: pointer;
  transition: background var(--transition-fast);
  
  &::after {
    content: '';
    position: absolute;
    top: 2px;
    left: 2px;
    width: 20px;
    height: 20px;
    background: white;
    border-radius: 50%;
    transition: transform var(--transition-fast);
  }
  
  &:checked {
    background: var(--accent-primary);
    
    &::after {
      transform: translateX(20px);
    }
  }
}

.danger-zone {
  border-color: rgba(239, 68, 68, 0.3);
  
  h3 {
    color: var(--accent-danger);
  }
  
  p {
    color: var(--text-muted);
    margin-bottom: 1rem;
  }
}
</style>
