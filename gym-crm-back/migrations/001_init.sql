CREATE TABLE IF NOT EXISTS schema_migrations (
    version VARCHAR(50) PRIMARY KEY,
    applied_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS admins (
    id SERIAL PRIMARY KEY,
    username VARCHAR(100) UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS terminals (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    ip VARCHAR(50) NOT NULL,
    port INTEGER DEFAULT 80,
    username VARCHAR(100) NOT NULL,
    password VARCHAR(100) NOT NULL,
    direction VARCHAR(10) NOT NULL CHECK (direction IN ('entry', 'exit')),
    active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS tariffs (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    duration_days INTEGER NOT NULL,
    max_visits_per_day INTEGER,
    price NUMERIC(10,2) NOT NULL,
    active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS clients (
    id SERIAL PRIMARY KEY,
    full_name VARCHAR(200) NOT NULL,
    phone VARCHAR(30),
    photo_path TEXT,
    card_number VARCHAR(100),
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS client_tariffs (
    id SERIAL PRIMARY KEY,
    client_id INTEGER REFERENCES clients(id),
    tariff_id INTEGER REFERENCES tariffs(id),
    start_date DATE NOT NULL,
    end_date DATE NOT NULL,
    paid_amount NUMERIC(10,2),
    payment_note TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS access_events (
    id BIGSERIAL PRIMARY KEY,
    client_id INTEGER REFERENCES clients(id),
    terminal_id INTEGER REFERENCES terminals(id),
    direction VARCHAR(10) NOT NULL CHECK (direction IN ('entry', 'exit')),
    auth_method VARCHAR(20),
    access_granted BOOLEAN NOT NULL,
    deny_reason TEXT,
    raw_event JSONB,
    event_time TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS refresh_tokens (
    id SERIAL PRIMARY KEY,
    admin_id INTEGER REFERENCES admins(id),
    token_hash TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
