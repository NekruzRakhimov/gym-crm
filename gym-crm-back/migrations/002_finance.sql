ALTER TABLE clients ADD COLUMN IF NOT EXISTS balance NUMERIC(10,2) NOT NULL DEFAULT 0;

CREATE TABLE IF NOT EXISTS transactions (
    id          BIGSERIAL PRIMARY KEY,
    client_id   INTEGER NOT NULL REFERENCES clients(id),
    type        VARCHAR(20) NOT NULL CHECK (type IN ('deposit', 'payment')),
    amount      NUMERIC(10,2) NOT NULL,
    description TEXT,
    client_tariff_id INTEGER REFERENCES client_tariffs(id),
    created_at  TIMESTAMPTZ DEFAULT NOW()
);
