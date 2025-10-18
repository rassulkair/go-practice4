CREATE TABLE IF NOT EXISTS users (
  id SERIAL PRIMARY KEY,
  name TEXT NOT NULL,
  email TEXT UNIQUE NOT NULL,
  balance NUMERIC(18,2) NOT NULL DEFAULT 0
);

INSERT INTO users (name, email, balance) VALUES
('Alice', 'alice@example.com', 100.00),
('Bob', 'bob@example.com', 50.00)
ON CONFLICT (email) DO NOTHING;
