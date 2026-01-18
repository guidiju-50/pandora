-- Create users table
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    role VARCHAR(50) NOT NULL DEFAULT 'researcher',
    active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    last_login_at TIMESTAMP WITH TIME ZONE
);

-- Create index on email
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);

-- Create index on role
CREATE INDEX IF NOT EXISTS idx_users_role ON users(role);

-- Insert default admin user
-- Email: admin@pandora.local
-- Password: Admin@123
INSERT INTO users (id, email, password_hash, name, role, active, created_at, updated_at)
VALUES (
    'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11',
    'admin@pandora.local',
    '$2a$10$p/SymUxwcQPh0wRzxpnVU.hoTxfu3NArmbb521g5bTYsrrBNh0vxW',
    'Administrator',
    'admin',
    TRUE,
    NOW(),
    NOW()
) ON CONFLICT (email) DO NOTHING;

-- Insert default researcher user
-- Email: researcher@pandora.local
-- Password: Research@123
INSERT INTO users (id, email, password_hash, name, role, active, created_at, updated_at)
VALUES (
    'b1eebc99-9c0b-4ef8-bb6d-6bb9bd380a22',
    'researcher@pandora.local',
    '$2a$10$2erv3HrOn9S9TKrDrVzoNe0Y6eN3kZqU/sZRQ.7NXrI9KrHTrq5jm',
    'Researcher',
    'researcher',
    TRUE,
    NOW(),
    NOW()
) ON CONFLICT (email) DO NOTHING;
