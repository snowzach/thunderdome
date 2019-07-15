SET client_encoding = 'UTF8';

-- account table
CREATE TABLE public.account (
  id TEXT PRIMARY KEY, -- type:id
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  updated_at TIMESTAMP WITH TIME ZONE,
  address TEXT NOT NULL DEFAULT '',
  balance BIGINT DEFAULT 0,
  pending_in BIGINT DEFAULT 0,
  pending_out BIGINT DEFAULT 0
);

-- Allow lookup by address
CREATE UNIQUE INDEX ix_account_address ON public.account USING btree(address);

-- Create the unknown account that can be used to handle unaccounted for transactions
INSERT INTO account (id, updated_at, address) VALUES ('internal:unknown', NOW(), 'unknown');

-- ledger types
CREATE TYPE ledger_status AS ENUM ('pending', 'completed', 'expired','failed');
CREATE TYPE ledger_type AS ENUM ('btc', 'lightning');
CREATE TYPE ledger_direction AS ENUM ('in', 'out');

-- ledger table
CREATE TABLE public.ledger (
  id TEXT NOT NULL,
  account_id TEXT NOT NULL,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  updated_at TIMESTAMP WITH TIME ZONE,
  expires_at TIMESTAMP WITH TIME ZONE,
  status ledger_status NOT NULL,
  type ledger_type NOT NULL,
  direction ledger_direction NOT NULL,
  value BIGINT DEFAULT 0,
  network_fee BIGINT DEFAULT 0,
  processing_fee BIGINT DEFAULT 0,
  add_index BIGINT DEFAULT 0,
  memo text DEFAULT '',
  request text DEFAULT '',
  error text DEFAULT '',
  PRIMARY KEY (id, direction)
);

ALTER TABLE ONLY public.ledger
  ADD CONSTRAINT fkey_ledger_account_id FOREIGN KEY (account_id) REFERENCES public.account(id) ON DELETE CASCADE;

-- Our default listing format
CREATE INDEX ix_ledger_account_id_status_updated_at ON public.ledger USING btree(account_id, status, updated_at);

-- Expiration
CREATE INDEX ix_ledger_status_expires_at ON public.ledger USING btree(status, expires_at);
