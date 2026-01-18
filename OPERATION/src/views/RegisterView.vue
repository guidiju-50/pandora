<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/store/auth'

const router = useRouter()
const authStore = useAuthStore()

const name = ref('')
const email = ref('')
const password = ref('')
const confirmPassword = ref('')
const loading = ref(false)
const error = ref('')

async function handleRegister() {
  if (!name.value || !email.value || !password.value) {
    error.value = 'Please fill in all fields'
    return
  }
  
  if (password.value !== confirmPassword.value) {
    error.value = 'Passwords do not match'
    return
  }
  
  if (password.value.length < 8) {
    error.value = 'Password must be at least 8 characters'
    return
  }
  
  loading.value = true
  error.value = ''
  
  const result = await authStore.register(name.value, email.value, password.value)
  
  loading.value = false
  
  if (result.success) {
    router.push('/')
  } else {
    error.value = result.error
  }
}
</script>

<template>
  <div class="register-page">
    <div class="register-card">
      <div class="register-header">
        <div class="logo">
          <svg width="48" height="48" viewBox="0 0 24 24" fill="none">
            <path d="M12 2L2 7L12 12L22 7L12 2Z" fill="url(#rg1)"/>
            <path d="M2 17L12 22L22 17" stroke="url(#rg2)" stroke-width="2"/>
            <path d="M2 12L12 17L22 12" stroke="url(#rg2)" stroke-width="2"/>
            <defs>
              <linearGradient id="rg1" x1="2" y1="2" x2="22" y2="12">
                <stop stop-color="#06b6d4"/>
                <stop offset="1" stop-color="#8b5cf6"/>
              </linearGradient>
              <linearGradient id="rg2" x1="2" y1="12" x2="22" y2="22">
                <stop stop-color="#06b6d4"/>
                <stop offset="1" stop-color="#8b5cf6"/>
              </linearGradient>
            </defs>
          </svg>
        </div>
        <h1>Create account</h1>
        <p>Join Pandora to start analyzing genomic data</p>
      </div>
      
      <form @submit.prevent="handleRegister" class="register-form">
        <div v-if="error" class="error-message">{{ error }}</div>
        
        <div class="form-group">
          <label class="form-label">Full Name</label>
          <input
            v-model="name"
            type="text"
            class="form-input"
            placeholder="Enter your name"
          />
        </div>
        
        <div class="form-group">
          <label class="form-label">Email</label>
          <input
            v-model="email"
            type="email"
            class="form-input"
            placeholder="Enter your email"
          />
        </div>
        
        <div class="form-group">
          <label class="form-label">Password</label>
          <input
            v-model="password"
            type="password"
            class="form-input"
            placeholder="Create a password"
          />
        </div>
        
        <div class="form-group">
          <label class="form-label">Confirm Password</label>
          <input
            v-model="confirmPassword"
            type="password"
            class="form-input"
            placeholder="Confirm your password"
          />
        </div>
        
        <button type="submit" class="btn btn--primary btn--lg" :disabled="loading" style="width: 100%">
          <span v-if="loading" class="animate-spin">⟳</span>
          <span v-else>Create Account</span>
        </button>
      </form>
      
      <div class="register-footer">
        <p>Already have an account? <router-link to="/login">Sign in</router-link></p>
      </div>
    </div>
  </div>
</template>

<style lang="scss" scoped>
.register-page {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 100vh;
  padding: 2rem;
}

.register-card {
  width: 100%;
  max-width: 400px;
  background: var(--bg-card);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-xl);
  padding: 2.5rem;
  animation: fadeIn 0.5s ease-out;
}

.register-header {
  text-align: center;
  margin-bottom: 2rem;
  
  .logo {
    display: flex;
    justify-content: center;
    margin-bottom: 1.5rem;
  }
  
  h1 {
    font-size: 1.75rem;
    margin-bottom: 0.5rem;
  }
  
  p {
    color: var(--text-muted);
  }
}

.register-form {
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
}

.error-message {
  padding: 0.75rem 1rem;
  background: rgba(239, 68, 68, 0.1);
  border: 1px solid rgba(239, 68, 68, 0.3);
  border-radius: var(--radius-md);
  color: var(--accent-danger);
  font-size: 0.875rem;
}

.register-footer {
  margin-top: 2rem;
  text-align: center;
  
  p {
    font-size: 0.875rem;
    color: var(--text-muted);
  }
  
  a {
    color: var(--accent-primary);
    font-weight: 500;
  }
}
</style>
